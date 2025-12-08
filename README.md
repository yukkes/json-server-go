# JSON Server Go [![Go](https://github.com/yukkes/json-server-go/actions/workflows/go.yml/badge.svg)](https://github.com/yukkes/json-server-go/actions/workflows/go.yml)

> **Note**: This is a Go port of the original [json-server](https://github.com/typicode/json-server). It aims to be a drop-in replacement with a single binary distribution.

Get a full fake REST API with __zero coding__ in __less than 30 seconds__ (seriously)

Created for front-end developers who need a quick back-end for prototyping and mocking.

## Table of contents

<!-- toc -->

- [Getting started](#getting-started)
- [Routes](#routes)
  * [Plural routes](#plural-routes)
  * [Singular routes](#singular-routes)
  * [Filter](#filter)
  * [Paginate](#paginate)
  * [Sort](#sort)
  * [Slice](#slice)
  * [Operators](#operators)
  * [Full-text search](#full-text-search)
  * [Relationships](#relationships)
  * [Database](#database)
  * [Homepage](#homepage)
- [Extras](#extras)
  * [Static file server](#static-file-server)
  * [Alternative port](#alternative-port)
  * [Access from anywhere](#access-from-anywhere)
  * [Add custom routes](#add-custom-routes)
  * [CLI usage](#cli-usage)
- [Performance Comparison](#performance-comparison)
- [License](#license)

<!-- tocstop -->

## Getting started

### Installation

```bash
git clone https://github.com/yukkes/json-server-go.git
cd json-server-go
go build -o json-server main.go
```

### Usage

Create a `db.json` file with some data

```json
{
  "posts": [
    { "id": 1, "title": "json-server", "author": "typicode" }
  ],
  "comments": [
    { "id": 1, "body": "some comment", "postId": 1 }
  ],
  "profile": { "name": "typicode" }
}
```

Start JSON Server

```bash
./json-server db.json
```

Or with watch mode:

```bash
./json-server --watch db.json
```

Now if you go to [http://localhost:3000/posts/1](http://localhost:3000/posts/1), you'll get

```json
{ "id": 1, "title": "json-server", "author": "typicode" }
```

Also when doing requests, it's good to know that:

- If you make POST, PUT, PATCH or DELETE requests, changes will be automatically and safely saved to `db.json`.
- Your request body JSON should be object enclosed, just like the GET output. (for example `{"name": "Foobar"}`)
- Id values are not mutable. Any `id` value in the body of your PUT or PATCH request will be ignored. Only a value set in a POST request will be respected, but only if not already taken.
- A POST, PUT or PATCH request should include a `Content-Type: application/json` header to use the JSON in the request body. Otherwise it will return a 2XX status code, but without changes being made to the data.

## Routes

Based on the previous `db.json` file, here are all the default routes. You can also add [other routes](#add-custom-routes) using `--routes`.

### Plural routes

```
GET    /posts
GET    /posts/1
POST   /posts
PUT    /posts/1
PATCH  /posts/1
DELETE /posts/1
```

### Singular routes

```
GET    /profile
POST   /profile
PUT    /profile
PATCH  /profile
```

### Filter

Use `.` to access deep properties

```
GET /posts?title=json-server&author=typicode
GET /posts?id=1&id=2
GET /comments?author.name=typicode
```

### Paginate

Use `_page` and optionally `_limit` to paginate returned data.

In the `Link` header you'll get `first`, `prev`, `next` and `last` links.


```
GET /posts?_page=7
GET /posts?_page=7&_limit=20
```

_10 items are returned by default_

### Sort

Add `_sort` and `_order` (ascending order by default)

```
GET /posts?_sort=views&_order=asc
GET /posts/1/comments?_sort=votes&_order=asc
```

For multiple fields, use the following format:

```
GET /posts?_sort=user,views&_order=desc,asc
```

### Slice

Add `_start` and `_end` or `_limit` (an `X-Total-Count` header is included in the response)

```
GET /posts?_start=20&_end=30
GET /posts/1/comments?_start=20&_end=30
GET /posts/1/comments?_start=20&_limit=10
```

_Works exactly as [Array.slice](https://developer.mozilla.org/en/docs/Web/JavaScript/Reference/Global_Objects/Array/slice) (i.e. `_start` is inclusive and `_end` exclusive)_

### Operators

Add `_gte` or `_lte` for getting a range

```
GET /posts?views_gte=10&views_lte=20
```

Add `_ne` to exclude a value

```
GET /posts?id_ne=1
```

Add `_like` to filter (RegExp supported)

```
GET /posts?title_like=server
```

### Full-text search

Add `q`

```
GET /posts?q=internet
```

### Relationships

To include children resources, add `_embed`

```
GET /posts?_embed=comments
GET /posts/1?_embed=comments
```

To include parent resource, add `_expand`

```
GET /comments?_expand=post
GET /comments/1?_expand=post
```

To get or create nested resources (by default one level, [add custom routes](#add-custom-routes) for more)

```
GET  /posts/1/comments
POST /posts/1/comments
```

### Database

```
GET /db
```

### Homepage

Returns default index file or serves `./public` directory

```
GET /
```

## Extras

### Static file server

You can use JSON Server to serve your HTML, JS and CSS, simply create a `./public` directory
or use `--static` to set a different static files directory.

```bash
mkdir public
echo 'hello world' > public/index.html
json-server db.json
```

```bash
json-server db.json --static ./some-other-dir
```

### Alternative port

You can start JSON Server on other ports with the `--port` flag:

```bash
$ json-server --watch db.json --port 3004
```

### Access from anywhere

You can access your fake API from anywhere using CORS and JSONP.

### Add custom routes

Create a `routes.json` file. Pay attention to start every route with `/`.

```json
{
  "/api/*": "/$1",
  "/:resource/:id/show": "/:resource/:id",
  "/posts/:category": "/posts?category=:category",
  "/articles\\?id=:id": "/posts/:id"
}
```

Start JSON Server with `--routes` option.

```bash
json-server db.json --routes routes.json
```

Now you can access resources using additional routes.

```sh
/api/posts # → /posts
/api/posts/1  # → /posts/1
/posts/1/show # → /posts/1
/posts/javascript # → /posts?category=javascript
/articles?id=1 # → /posts/1
```

### CLI usage

```
json-server [options] <source>

Options:
  --port, -p         Set port                                    [default: 3000]
  --host, -H         Set host                             [default: "localhost"]
  --watch, -w        Watch file(s)                                     [boolean]
  --routes, -r       Path to routes file
  --static, -s       Set static files directory
  --read-only, -ro   Allow only GET requests                           [boolean]
  --no-cors, -nc     Disable Cross-Origin Resource Sharing             [boolean]
  --delay, -d        Add delay to responses (ms)
  --id, -i           Set database id property (e.g. _id)         [default: "id"]
  --quiet, -q        Suppress log messages from output                 [boolean]
  --help, -h         Show help                                         [boolean]
  --version, -v      Show version number                               [boolean]

Examples:
  json-server db.json
  json-server db.json --quiet # Disable access logs
```

## Feature Comparison: Node.js vs Go Port

This section outlines the feature differences between the original Node.js `json-server` and this Go port.

### Feature Comparison Table

| Feature Category | Feature | Node.js (Original) | Go Port | Notes |
| :--- | :--- | :---: | :---: | :--- |
| **CLI Options** | Port / Host | ✅ | ✅ | |
| | Watch Mode (`--watch`) | ✅ | ✅ | |
| | Custom Routes (`--routes`) | ✅ | ✅ | |
| | Middlewares (`--middlewares`) | ✅ | ❌ | JS specific. |
| | Static Files (`--static`) | ✅ | ✅ | |
| | Read Only (`--read-only`) | ✅ | ✅ | |
| | No CORS (`--no-cors`) | ✅ | ✅ | |
| | Delay (`--delay`) | ✅ | ✅ | |
| | ID Property (`--id`) | ✅ | ✅ | |
| **Routes** | Plural Routes (CRUD) | ✅ | ✅ | |
| | Singular Routes | ✅ | ✅ | |
| | Nested Routes (e.g. `/posts/1/comments`) | ✅ | ✅ | |
| | Database Route (`/db`) | ✅ | ✅ | |
| **Filtering** | Simple Equality | ✅ | ✅ | |
| | Deep Properties (e.g. `author.name`) | ✅ | ✅ | |
| | Operators (`_gte`, `_lte`, `_ne`, `_like`) | ✅ | ✅ | |
| **Pagination** | `_page`, `_limit` | ✅ | ✅ | |
| | Link Headers | ✅ | ✅ | |
| **Sorting** | `_sort`, `_order` | ✅ | ✅ | |
| | Multiple Fields Sorting | ✅ | ✅ | |
| **Slice** | `_start`, `_end`, `_limit` | ✅ | ✅ | |
| | `X-Total-Count` Header | ✅ | ✅ | |
| **Search** | Full-text search (`q`) | ✅ | ✅ | |
| **Relationships** | `_embed` (Children) | ✅ | ✅ | |
| | `_expand` (Parent) | ✅ | ✅ | |
| **Data Source** | JSON File | ✅ | ✅ | |
| | JS File (Random Data) | ✅ | ❌ | JS specific. |
| | Remote URL | ✅ | ❌ | |

### Detailed Missing Features in Go Port

#### 1. CLI Configuration & Customization
Some CLI flags are not yet implemented or are specific to the Node.js ecosystem:
- **`--middlewares`:** Cannot load external middleware scripts (Node.js specific).

#### 2. Data Generation
The Go port only accepts a JSON file as input. It cannot execute a JavaScript file to generate data programmatically, nor can it load data from a remote URL.

#### 3. ID Generation
The Go port now supports smart ID generation matching Node.js behavior:
- If the collection is empty, it returns `1`.
- If the collection has items with numeric IDs, it finds the max ID and increments it.
- Otherwise (string IDs), it generates a random 7-character string ID (NanoID style).

## Performance Comparison

This section provides a performance comparison between the original Node.js json-server and this Go port, based on ApacheBench (ab) tests.

**Test Environment:**
- CPU: AMD Ryzen AI 7 350
- Test Command: `ab -n 10000 -c 10 http://localhost:3000/posts`
- Endpoint: `/posts`
- Requests: 10,000
- Concurrency: 10

**Performance Results:**

| Version                  | Requests per second | Time per request (ms) | Notes                  |
|--------------------------|---------------------|-----------------------|------------------------|
| Original (Node.js)       | 1642.89            | 6.09                 | Access logging enabled |
| Go Port (with logging)   | 14494.62           | 0.69                 | Access logging enabled |
| Go Port (without logging)| 22552.47           | 0.44                 | Access logging disabled |

The Go port demonstrates significantly higher performance compared to the original Node.js version, with the version without access logging being approximately 13.7 times faster in terms of requests per second.

## License

MIT
