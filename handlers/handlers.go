package handlers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"json-server-go/db"

	"github.com/gorilla/mux"
)

type Handler struct {
	DB      *db.Database
	IdField string
}

func New(d *db.Database, idField string) *Handler {
	if idField == "" {
		idField = "id"
	}
	return &Handler{DB: d, IdField: idField}
}

// --- Plural Routes ---

func (h *Handler) PluralList(w http.ResponseWriter, r *http.Request, resourceName string) {
	data, ok := h.DB.Get(resourceName)
	if !ok {
		http.NotFound(w, r)
		return
	}

	list, ok := data.([]interface{})
	if !ok {
		http.Error(w, "Resource is not a list", http.StatusInternalServerError)
		return
	}

	// Clone list to avoid modifying DB data during processing (e.g. _embed)
	// We only need to clone the slice structure initially, but if we modify elements (embed), we need deep clone.
	// For performance, we'll clone elements only when needed or use a deep clone strategy if _embed/_expand is used.
	// For now, let's work with the slice.

	// Filtering, Sorting, Pagination, Embed, Expand
	result, totalCount := h.processList(list, r.URL.Query(), r)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.Itoa(totalCount))

	// Link Header
	h.setLinkHeader(w, r.URL, totalCount, r.URL.Query())

	json.NewEncoder(w).Encode(result)
}

func (h *Handler) PluralGet(w http.ResponseWriter, r *http.Request, resourceName string) {
	vars := mux.Vars(r)
	id := vars["id"]

	data, ok := h.DB.Get(resourceName)
	if !ok {
		http.NotFound(w, r)
		return
	}

	list, ok := data.([]interface{})
	if !ok {
		http.Error(w, "Resource is not a list", http.StatusInternalServerError)
		return
	}

	item := h.findById(list, id)
	if item == nil {
		http.NotFound(w, r)
		return
	}

	// Clone item to avoid modifying DB
	item = deepClone(item)

	// Embed and Expand
	query := r.URL.Query()
	if embeds, ok := query["_embed"]; ok {
		for _, embedRes := range embeds {
			h.embed(item, resourceName, embedRes)
		}
	}
	if expands, ok := query["_expand"]; ok {
		for _, expandRes := range expands {
			h.expand(item, expandRes)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func (h *Handler) PluralCreate(w http.ResponseWriter, r *http.Request, resourceName string) {
	var newItem map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&newItem); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate ID if not present
	if _, ok := newItem[h.IdField]; !ok {
		list := h.DB.Data[resourceName].([]interface{})
		newItem[h.IdField] = h.generateId(list)
	}

	h.DB.Data[resourceName] = append(h.DB.Data[resourceName].([]interface{}), newItem)
	h.DB.Save()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newItem)
}

func (h *Handler) PluralUpdate(w http.ResponseWriter, r *http.Request, resourceName string) {
	vars := mux.Vars(r)
	id := vars["id"]

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	list := h.DB.Data[resourceName].([]interface{})
	idx := h.findIndexById(list, id)

	if idx == -1 {
		http.NotFound(w, r)
		return
	}

	// PUT replaces the object, but keeps the ID
	updates[h.IdField] = id // Ensure ID is preserved/set
	list[idx] = updates
	h.DB.Data[resourceName] = list
	h.DB.Save()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updates)
}

func (h *Handler) PluralPatch(w http.ResponseWriter, r *http.Request, resourceName string) {
	vars := mux.Vars(r)
	id := vars["id"]

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	list := h.DB.Data[resourceName].([]interface{})
	idx := h.findIndexById(list, id)

	if idx == -1 {
		http.NotFound(w, r)
		return
	}

	currentItem := list[idx].(map[string]interface{})
	for k, v := range updates {
		if k != h.IdField { // Don't update ID
			currentItem[k] = v
		}
	}

	list[idx] = currentItem
	h.DB.Data[resourceName] = list
	h.DB.Save()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(currentItem)
}

func (h *Handler) PluralDelete(w http.ResponseWriter, r *http.Request, resourceName string) {
	vars := mux.Vars(r)
	id := vars["id"]

	list := h.DB.Data[resourceName].([]interface{})
	idx := h.findIndexById(list, id)

	if idx == -1 {
		http.NotFound(w, r)
		return
	}

	// Remove from slice
	h.DB.Data[resourceName] = append(list[:idx], list[idx+1:]...)
	h.DB.Save()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

// --- Singular Routes ---

func (h *Handler) SingularGet(w http.ResponseWriter, r *http.Request, resourceName string) {
	data, ok := h.DB.Get(resourceName)
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) SingularUpdate(w http.ResponseWriter, r *http.Request, resourceName string) {
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodPut {
		h.DB.Set(resourceName, updates)
	} else { // PATCH
		current, _ := h.DB.Get(resourceName)
		currentMap, ok := current.(map[string]interface{})
		if !ok {
			// If it wasn't a map (maybe null), just set it
			h.DB.Set(resourceName, updates)
		} else {
			for k, v := range updates {
				currentMap[k] = v
			}
			h.DB.Set(resourceName, currentMap)
		}
	}
	h.DB.Save()

	newData, _ := h.DB.Get(resourceName)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newData)
}

// --- Helpers ---

func (h *Handler) processList(list []interface{}, query map[string][]string, r *http.Request) ([]interface{}, int) {
	// Shallow copy list to avoid modifying the original slice structure
	result := make([]interface{}, len(list))
	copy(result, list)

	// 1. Filtering
	for key, values := range query {
		if strings.HasPrefix(key, "_") || key == "q" {
			continue
		}
		result = filter(result, key, values)
	}

	// 2. Full-text search (q)
	if q, ok := query["q"]; ok && len(q) > 0 {
		result = fullTextSearch(result, q[0])
	}

	// 3. Sorting (_sort, _order)
	if sortKeys, ok := query["_sort"]; ok && len(sortKeys) > 0 {
		orderStr := "asc"
		if orders, ok := query["_order"]; ok && len(orders) > 0 {
			orderStr = orders[0]
		}

		keys := strings.Split(sortKeys[0], ",")
		orders := strings.Split(orderStr, ",")

		result = sortList(result, keys, orders)
	}

	// Total count after filtering/sorting but before pagination
	totalCount := len(result)

	// 4. Pagination (_page, _limit) or Slicing (_start, _end, _limit)
	if pages, ok := query["_page"]; ok && len(pages) > 0 {
		page, _ := strconv.Atoi(pages[0])
		limit := 10
		if limits, ok := query["_limit"]; ok && len(limits) > 0 {
			limit, _ = strconv.Atoi(limits[0])
		}
		start := (page - 1) * limit
		end := start + limit
		if start < 0 {
			start = 0
		}
		if start > len(result) {
			start = len(result)
		}
		if end > len(result) {
			end = len(result)
		}
		result = result[start:end]
	} else {
		start := 0
		end := len(result)

		if starts, ok := query["_start"]; ok && len(starts) > 0 {
			start, _ = strconv.Atoi(starts[0])
		}

		if ends, ok := query["_end"]; ok && len(ends) > 0 {
			end, _ = strconv.Atoi(ends[0])
		} else if limits, ok := query["_limit"]; ok && len(limits) > 0 {
			limit, _ := strconv.Atoi(limits[0])
			end = start + limit
		}

		if start < 0 {
			start = 0
		}
		if start > len(result) {
			start = len(result)
		}
		if end > len(result) {
			end = len(result)
		}
		if end < start {
			end = start
		}

		result = result[start:end]
	}

	// 5. Embed and Expand
	// We need to know the resource name to infer relationships.
	// However, processList is generic. We can pass resourceName or infer it?
	// The caller PluralList knows the resourceName.
	// But wait, processList signature doesn't have resourceName.
	// Let's extract resourceName from URL or pass it.
	// URL path is /resourceName usually.
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) > 0 {
		resourceName := pathParts[0] // Simple assumption

		_, hasEmbed := query["_embed"]
		_, hasExpand := query["_expand"]

		if hasEmbed || hasExpand {
			// Deep clone only the items we are about to modify
			for i, item := range result {
				result[i] = deepClone(item)
			}

			if hasEmbed {
				for _, embedRes := range query["_embed"] {
					for _, item := range result {
						h.embed(item, resourceName, embedRes)
					}
				}
			}
			if hasExpand {
				for _, expandRes := range query["_expand"] {
					for _, item := range result {
						h.expand(item, expandRes)
					}
				}
			}
		}
	}

	return result, totalCount
}

func filter(list []interface{}, key string, values []string) []interface{} {
	var filtered []interface{}

	// Check for operators
	operator := ""
	cleanKey := key
	if strings.HasSuffix(key, "_gte") {
		operator = "_gte"
		cleanKey = strings.TrimSuffix(key, "_gte")
	} else if strings.HasSuffix(key, "_lte") {
		operator = "_lte"
		cleanKey = strings.TrimSuffix(key, "_lte")
	} else if strings.HasSuffix(key, "_ne") {
		operator = "_ne"
		cleanKey = strings.TrimSuffix(key, "_ne")
	} else if strings.HasSuffix(key, "_like") {
		operator = "_like"
		cleanKey = strings.TrimSuffix(key, "_like")
	}

	for _, item := range list {
		val := getValueByPath(item, cleanKey)
		if val == nil {
			continue
		}

		match := false
		for _, v := range values {
			if checkMatch(val, v, operator) {
				match = true
				break
			}
		}
		if match {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func getValueByPath(item interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	current := item
	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			if val, exists := m[part]; exists {
				current = val
			} else {
				return nil
			}
		} else {
			return nil
		}
	}
	return current
}

func checkMatch(val interface{}, queryVal string, operator string) bool {
	strVal := fmt.Sprintf("%v", val)

	switch operator {
	case "_gte":
		return compare(val, queryVal) >= 0
	case "_lte":
		return compare(val, queryVal) <= 0
	case "_ne":
		return strVal != queryVal
	case "_like":
		matched, _ := regexp.MatchString("(?i)"+queryVal, strVal)
		return matched
	default:
		return strVal == queryVal
	}
}

func compare(val interface{}, queryVal string) int {
	// Try number comparison
	f1, ok1 := toFloat(val)
	f2, err := strconv.ParseFloat(queryVal, 64)

	if ok1 && err == nil {
		if f1 > f2 {
			return 1
		}
		if f1 < f2 {
			return -1
		}
		return 0
	}

	// String comparison
	s1 := fmt.Sprintf("%v", val)
	if s1 > queryVal {
		return 1
	}
	if s1 < queryVal {
		return -1
	}
	return 0
}

func toFloat(v interface{}) (float64, bool) {
	switch i := v.(type) {
	case float64:
		return i, true
	case int:
		return float64(i), true
	case string:
		f, err := strconv.ParseFloat(i, 64)
		return f, err == nil
	}
	return 0, false
}

func fullTextSearch(list []interface{}, query string) []interface{} {
	var filtered []interface{}
	query = strings.ToLower(query)
	for _, item := range list {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		match := false
		// Recursive search could be better, but flat search for now
		for _, v := range obj {
			if strVal, ok := v.(string); ok {
				if strings.Contains(strings.ToLower(strVal), query) {
					match = true
					break
				}
			}
		}
		if match {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func sortList(list []interface{}, keys []string, orders []string) []interface{} {
	sort.SliceStable(list, func(i, j int) bool {
		obj1 := list[i].(map[string]interface{})
		obj2 := list[j].(map[string]interface{})

		for k, key := range keys {
			order := "asc"
			if k < len(orders) {
				order = strings.ToLower(orders[k])
			}

			v1 := getValueByPath(obj1, key)
			v2 := getValueByPath(obj2, key)

			if v1 == nil && v2 == nil {
				continue
			}
			if v1 == nil {
				return false
			} // nil last
			if v2 == nil {
				return true
			}

			cmp := compare(v1, fmt.Sprintf("%v", v2))
			if cmp != 0 {
				if order == "desc" {
					return cmp > 0
				}
				return cmp < 0
			}
		}
		return false
	})
	return list
}

func (h *Handler) findById(list []interface{}, id string) interface{} {
	for _, item := range list {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if fmt.Sprintf("%v", obj[h.IdField]) == id {
			return item
		}
	}
	return nil
}

func (h *Handler) findIndexById(list []interface{}, id string) int {
	for i, item := range list {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if fmt.Sprintf("%v", obj[h.IdField]) == id {
			return i
		}
	}
	return -1
}

func (h *Handler) generateId(list []interface{}) interface{} {
	if len(list) == 0 {
		return 1
	}

	// Find max ID
	var maxId float64
	var hasNumericId bool

	for _, item := range list {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if val, ok := obj[h.IdField]; ok {
			if f, ok := toFloat(val); ok {
				if !hasNumericId || f > maxId {
					maxId = f
					hasNumericId = true
				}
			}
		}
	}

	if hasNumericId {
		return maxId + 1
	}

	return generateRandomId()
}

func generateRandomId() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 7)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

func deepClone(src interface{}) interface{} {
	b, _ := json.Marshal(src)
	var dest interface{}
	json.Unmarshal(b, &dest)
	return dest
}

// --- Relationship Helpers ---

func (h *Handler) embed(item interface{}, resourceName string, embedResource string) {
	// item is the parent (e.g. post)
	// embedResource is the child (e.g. comments)
	// We need to find comments where comment.postId == item.id

	itemMap, ok := item.(map[string]interface{})
	if !ok {
		return
	}

	id, ok := itemMap[h.IdField]
	if !ok {
		return
	}

	// Find the child collection
	childData, ok := h.DB.Get(embedResource)
	if !ok {
		return
	}

	childList, ok := childData.([]interface{})
	if !ok {
		return
	}

	// Infer foreign key: singular(resourceName) + "Id"
	// e.g. posts -> postId
	singularName := strings.TrimSuffix(resourceName, "s") // Naive singularization
	foreignKey := singularName + "Id"

	var embedded []interface{}
	for _, child := range childList {
		childMap, ok := child.(map[string]interface{})
		if !ok {
			continue
		}

		if val, exists := childMap[foreignKey]; exists {
			// Compare IDs
			if fmt.Sprintf("%v", val) == fmt.Sprintf("%v", id) {
				embedded = append(embedded, child)
			}
		}
	}

	itemMap[embedResource] = embedded
}

func (h *Handler) expand(item interface{}, expandResource string) {
	// item is the child (e.g. comment)
	// expandResource is the parent (e.g. post)
	// We need to find post where post.id == comment.postId

	itemMap, ok := item.(map[string]interface{})
	if !ok {
		return
	}

	// Infer foreign key: expandResource + "Id"
	// e.g. post -> postId
	foreignKey := expandResource + "Id"

	foreignId, ok := itemMap[foreignKey]
	if !ok {
		return
	}

	// Find the parent collection
	// We need to guess the plural name of the parent resource.
	// e.g. post -> posts
	parentCollectionName := expandResource + "s" // Naive pluralization

	parentData, ok := h.DB.Get(parentCollectionName)
	if !ok {
		return
	}

	parentList, ok := parentData.([]interface{})
	if !ok {
		return
	}

	parent := h.findById(parentList, fmt.Sprintf("%v", foreignId))
	if parent != nil {
		itemMap[expandResource] = parent
	}
}

func (h *Handler) setLinkHeader(w http.ResponseWriter, u *url.URL, totalCount int, query url.Values) {
	pageStr := query.Get("_page")
	if pageStr == "" {
		return
	}

	page, _ := strconv.Atoi(pageStr)
	limit := 10
	if l := query.Get("_limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}

	lastPage := int(math.Ceil(float64(totalCount) / float64(limit)))
	if lastPage == 0 {
		lastPage = 1
	}

	links := []string{}

	// Helper to create link
	createLink := func(p int, rel string) string {
		q := u.Query()
		q.Set("_page", strconv.Itoa(p))
		u.RawQuery = q.Encode()
		return fmt.Sprintf("<%s>; rel=\"%s\"", u.String(), rel)
	}

	if page > 1 {
		links = append(links, createLink(1, "first"))
		links = append(links, createLink(page-1, "prev"))
	}

	if page < lastPage {
		links = append(links, createLink(page+1, "next"))
		links = append(links, createLink(lastPage, "last"))
	}

	if len(links) > 0 {
		w.Header().Set("Link", strings.Join(links, ", "))
	}
}
