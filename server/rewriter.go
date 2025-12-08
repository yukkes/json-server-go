package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type Rewriter struct {
	Rules []RewriteRule
}

type RewriteRule struct {
	Pattern *regexp.Regexp
	Target  string
}

func NewRewriter(filePath string) (*Rewriter, error) {
	if filePath == "" {
		return nil, nil
	}

	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var routes map[string]string
	if err := json.Unmarshal(file, &routes); err != nil {
		return nil, err
	}

	rewriter := &Rewriter{}
	for pattern, target := range routes {
		// Convert express-style pattern to regex
		// /api/* -> ^/api/(.*)$
		// /:resource/:id -> ^/([^/]+)/([^/]+)$

		regexPattern := "^" + pattern

		// Handle wildcards
		regexPattern = strings.ReplaceAll(regexPattern, "*", "(.*)")

		// Handle named parameters :name
		// We need to be careful not to replace :name in target if it exists there (it shouldn't usually)
		// But here we are processing the key (pattern)

		// Regex to find :param
		reParam := regexp.MustCompile(`:([a-zA-Z0-9_]+)`)

		// Replace :param with ([^/]+)
		regexPattern = reParam.ReplaceAllString(regexPattern, `([^/]+)`)

		regexPattern += "$"

		re, err := regexp.Compile(regexPattern)
		if err != nil {
			log.Printf("Invalid route pattern: %s", pattern)
			continue
		}

		// Process target to use $1, $2 etc.
		// If the pattern had params, we need to map them.
		// However, standard json-server rewriter often relies on order or explicit $1.
		// If user writes /:resource/:id -> /:resource/:id, we need to convert target :resource to $1, :id to $2.

		// Find all params in original pattern to know their order
		params := reParam.FindAllStringSubmatch(pattern, -1)

		processedTarget := target
		for i, p := range params {
			paramName := p[0] // :resource
			// Replace :resource with $1, $2...
			// Note: Go uses ${1} for replacement
			processedTarget = strings.ReplaceAll(processedTarget, paramName, fmt.Sprintf("${%d}", i+1))
		}

		// Also handle explicit $1, $2 in target if user used wildcard in pattern
		// Go uses $1, but let's ensure it works.

		rewriter.Rules = append(rewriter.Rules, RewriteRule{
			Pattern: re,
			Target:  processedTarget,
		})
	}

	return rewriter, nil
}

func (rw *Rewriter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, rule := range rw.Rules {
			if rule.Pattern.MatchString(r.URL.Path) {
				originalPath := r.URL.Path
				newPath := rule.Pattern.ReplaceAllString(originalPath, rule.Target)

				// If target contains query params (e.g. /posts?category=$1), we need to handle it
				if strings.Contains(newPath, "?") {
					parts := strings.SplitN(newPath, "?", 2)
					r.URL.Path = parts[0]

					// Strip trailing slash if it exists (e.g. /comments/?... -> /comments/)
					if strings.HasSuffix(r.URL.Path, "/") && len(r.URL.Path) > 1 {
						r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
					}

					// Merge query params
					newQuery := parts[1]
					// Parse new query
					// We need to parse it manually or use url.ParseQuery
					// But simpler is to just append to RawQuery if empty, or join
					if r.URL.RawQuery == "" {
						r.URL.RawQuery = newQuery
					} else {
						r.URL.RawQuery += "&" + newQuery
					}
				} else {
					r.URL.Path = newPath
				}

				log.Printf("Rewrote %s to %s", originalPath, r.URL.String())
				break // Apply first matching rule
			}
		}
		next.ServeHTTP(w, r)
	})
}
