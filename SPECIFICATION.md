# JSON Server Architecture and Specification Documentation

This document provides a detailed overview of the `json-server` codebase to facilitate refactoring into another programming language. It covers the architecture, core components, data handling, routing logic, and CLI implementation.

## 1. Architecture Overview

`json-server` is built on top of **Express.js**. It serves a REST API based on a JSON database (or JS object). The core philosophy is to provide a full fake REST API with zero coding.

The project is divided into two main parts:
1.  **Server Library (`src/server`)**: The core logic that creates the Express app, sets up the router, and handles requests.
2.  **CLI (`src/cli`)**: The command-line interface that parses arguments, loads data, and starts the server.

## 2. Core Components

### 2.1 Server (`src/server/index.js`)

The server module exports the following functions:
-   `create()`: Returns a new Express application instance.
-   `defaults(options)`: Returns an array of default middlewares.
-   `router(source, options)`: Returns the Express router configured with the database.
-   `rewriter(rules)`: Returns a middleware for URL rewriting.
-   `bodyParser`: The body-parser middleware.

### 2.2 Defaults (`src/server/defaults.js`)

The `defaults` function configures standard middlewares. It accepts an options object:

-   `static`: Path to static files (default: `public` directory).
-   `logger`: Enable/disable logger (default: `true`).
-   `bodyParser`: Enable/disable body parser (default: `true`).
-   `noCors`: Disable CORS (default: `false`).
-   `readOnly`: Accept only GET requests (default: `false`).
-   `noGzip`: Disable GZIP compression (default: `false`).

**Included Middlewares:**
1.  `compression`: GZIP compression.
2.  `cors`: Cross-Origin Resource Sharing (allows all origins).
3.  `errorhandler`: Error handler (development mode only).
4.  `express.static`: Serves static files.
5.  `morgan`: HTTP request logger.
6.  **No-Cache Headers**: Sets `Cache-Control: no-cache`, `Pragma: no-cache`, `Expires: -1` for IE compatibility.
7.  **Read-Only Check**: If `readOnly` is true, returns 403 for non-GET requests.
8.  `bodyParser`: Parses JSON and URL-encoded bodies.

### 2.3 Router (`src/server/router/index.js`)

The router is the heart of `json-server`. It initializes the database and dynamically creates routes based on the data structure.

**Initialization:**
1.  **Database**: Uses `lowdb`. Supports `FileSync` (for JSON files) and `Memory` (for objects/URLs).
2.  **Mixins**: Adds `lodash-id` and custom mixins (`src/server/mixins.js`) to `lowdb` for ID generation and other utilities.
3.  **Middleware**: Adds `method-override` and `body-parser`.
4.  **Validation**: Validates that the database state is an object.
5.  **Exposed Endpoints**:
    -   `GET /db`: Returns the full database state.
6.  **Nested Routes**: Handles `/:parent/:parentId/:resource` via `nested.js`.
7.  **Dynamic Routes**: Iterates over the keys in the database:
    -   **Array values**: Treated as **Plural Resources** (e.g., `/posts`). Handled by `plural.js`.
    -   **Object values**: Treated as **Singular Resources** (e.g., `/profile`). Handled by `singular.js`.
8.  **Error Handling**:
    -   404 Not Found: If `res.locals.data` is not set.
    -   500 Internal Server Error: Catches exceptions.

## 3. Data Handling

-   **Library**: `lowdb` is used for data management.
-   **Persistence**: Changes are persisted to the JSON file (if used) via `src/server/router/write.js`. This middleware calls `db.write()` after modification requests (POST, PUT, PATCH, DELETE).
-   **ID Generation**: `lodash-id` is used to generate unique IDs (UUIDs by default) for new items.
-   **In-Memory**: If the source is a URL or a JS object, the database is in-memory and changes are not persisted to disk (unless snapshots are used).

## 4. Routing Logic

### 4.1 Plural Routes (`src/server/router/plural.js`)

Handles arrays of objects. Supports full CRUD operations.

**Endpoints:**
-   `GET /resource`: List items.
-   `GET /resource/:id`: Get item by ID.
-   `POST /resource`: Create new item.
-   `PUT /resource/:id`: Replace item.
-   `PATCH /resource/:id`: Update item (partial).
-   `DELETE /resource/:id`: Delete item.

**Features (GET /resource):**
-   **Filtering**: Query parameters match properties (e.g., `?title=json-server`).
    -   Deep properties: `?author.name=typicode`.
    -   Operators:
        -   `_gte`, `_lte`: Range (Greater/Less Than or Equal).
        -   `_ne`: Not Equal.
        -   `_like`: RegExp match.
-   **Full-text Search**: `?q=term`. Searches all properties.
-   **Sorting**: `?_sort=field&_order=asc|desc`.
-   **Pagination**: `?_page=1&_limit=10`. Adds `Link` header.
-   **Slicing**: `?_start=0&_end=10` or `?_start=0&_limit=10`. Adds `X-Total-Count` header.
-   **Relationships**:
    -   `_embed`: Include children resources (e.g., `/posts?_embed=comments`). Looks for `resourceId` in other collections.
    -   `_expand`: Include parent resource (e.g., `/comments?_expand=post`). Looks for `resourceId` in the current item.

### 4.2 Singular Routes (`src/server/router/singular.js`)

Handles single objects.

**Endpoints:**
-   `GET /resource`: Get the object.
-   `POST /resource`: Create/Overwrite.
-   `PUT /resource`: Update/Overwrite.
-   `PATCH /resource`: Update (partial).

### 4.3 Nested Routes (`src/server/router/nested.js`)

Rewrites URLs to support nested resources.
-   Pattern: `/:resource/:id/:nested`
-   Action: Rewrites to `/:nested` and adds `?resourceId=:id` to the query (for GET) or body (for POST).
-   Example: `/posts/1/comments` -> `/comments?postId=1`.

### 4.4 Custom Routes (`src/server/rewriter.js`)

Uses `express-urlrewrite` to support custom routing rules defined in a `routes.json` file.
-   Example: `"/api/*": "/$1"`

## 5. CLI Implementation (`src/cli`)

### 5.1 Argument Parsing (`src/cli/index.js`)

Uses `yargs` to parse command-line arguments.
-   `--port`, `-p`: Port (default: 3000).
-   `--host`, `-H`: Host (default: localhost).
-   `--watch`, `-w`: Watch mode.
-   `--routes`, `-r`: Path to routes file.
-   `--middlewares`, `-m`: Path to middleware files.
-   `--static`, `-s`: Static files directory.
-   `--read-only`, `--ro`: Read-only mode.
-   `--no-cors`, `--nc`: Disable CORS.
-   `--no-gzip`, `--ng`: Disable GZIP.
-   `--snapshots`, `-S`: Snapshots directory.
-   `--delay`, `-d`: Artificial delay.
-   `--id`, `-i`: ID property name (default: `id`).

### 5.2 Startup Logic (`src/cli/run.js`)

1.  **Load Data**: Reads the source (JSON file, JS file, or URL).
2.  **Load Config**: Reads `json-server.json` or arguments.
3.  **Create App**: Calls `createApp` which assembles the server, defaults, rewriter, middlewares, and router.
4.  **Start Server**: Listens on the specified port.
5.  **Watch Mode**:
    -   Watches the directory of the source file.
    -   Reloads the server if the file changes.
    -   Note: `lowdb` writes are atomic, so it watches the directory.
6.  **Snapshots**: Listens for `s` key on `stdin` to save a snapshot of the current database state.

## 6. Key Dependencies

-   `express`: Web framework.
-   `lowdb`: JSON database.
-   `lodash`: Utility library (heavy usage for data manipulation).
-   `pluralize`: Singular/Plural conversion.
-   `yargs`: CLI argument parsing.
-   `cors`, `compression`, `morgan`, `errorhandler`: Express middlewares.

## 7. Refactoring Considerations

When refactoring to another language, consider the following:

1.  **Dynamic Routing**: The core feature is generating routes based on the data structure. The new implementation must be able to inspect the data and register routes dynamically.
2.  **Query Language**: The filtering, sorting, and slicing logic (`_gte`, `_like`, `_embed`, etc.) is specific to `json-server` and relies heavily on `lodash`. This logic needs to be reimplemented.
3.  **Data Persistence**: You need a mechanism to read/write JSON files atomically if you want to support the file-based database feature.
4.  **Middleware System**: If the target language has a web framework with middleware support (like Go's `net/http` or Python's `Flask`/`Django`), map the existing middlewares (CORS, GZIP, Static) to the new framework's equivalents.
5.  **CLI**: Reimplement the CLI arguments and the watch mode functionality.
