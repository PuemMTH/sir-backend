package handler

import (
	"net/http"
)

const openapiJSON = `{
  "openapi": "3.0.3",
  "info": {
    "title": "sir-backend API",
    "version": "1.1.0"
  },
  "components": {
    "securitySchemes": {
      "bearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT"
      }
    },
    "schemas": {
      "Error": {
        "type": "object",
        "properties": {
          "error": { "type": "string" }
        }
      },
      "User": {
        "type": "object",
        "properties": {
          "id":         { "type": "string" },
          "email":      { "type": "string" },
          "role":       { "type": "string", "enum": ["user", "admin"] },
          "created_at": { "type": "integer" }
        }
      },
      "Note": {
        "type": "object",
        "properties": {
          "id":         { "type": "string" },
          "user_id":    { "type": "string" },
          "title":      { "type": "string" },
          "content":    { "type": "string" },
          "created_at": { "type": "integer" },
          "updated_at": { "type": "integer" }
        }
      },
      "LatexFile": {
        "type": "object",
        "properties": {
          "id":         { "type": "string" },
          "user_id":    { "type": "string" },
          "name":       { "type": "string" },
          "r2_key":     { "type": "string" },
          "engine":     { "type": "string", "enum": ["lualatex", "pdflatex", "xelatex"] },
          "created_at": { "type": "integer" },
          "updated_at": { "type": "integer" }
        }
      },
      "LatexFileWithContent": {
        "type": "object",
        "properties": {
          "id":         { "type": "string" },
          "user_id":    { "type": "string" },
          "name":       { "type": "string" },
          "engine":     { "type": "string", "enum": ["lualatex", "pdflatex", "xelatex"] },
          "content":    { "type": "string" },
          "created_at": { "type": "integer" },
          "updated_at": { "type": "integer" }
        }
      },
      "TokenResponse": {
        "type": "object",
        "properties": {
          "access_token":  { "type": "string" },
          "token_type":    { "type": "string", "example": "Bearer" },
          "expires_in":    { "type": "integer", "example": 3600 },
          "refresh_token": { "type": "string" },
          "scope":         { "type": "string" }
        }
      }
    }
  },
  "paths": {
    "/setup": {
      "post": {
        "tags": ["Setup"],
        "summary": "Initialize admin user and OAuth client",
        "parameters": [
          {
            "name": "secret",
            "in": "query",
            "required": true,
            "schema": { "type": "string" }
          }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["admin_email", "admin_password", "client_id", "client_secret"],
                "properties": {
                  "admin_email":    { "type": "string" },
                  "admin_password": { "type": "string" },
                  "client_id":     { "type": "string" },
                  "client_secret": { "type": "string" },
                  "client_name":   { "type": "string", "default": "Default Client" }
                }
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Created",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "admin_id":    { "type": "string" },
                    "admin_email": { "type": "string" },
                    "client_id":   { "type": "string" },
                    "client_name": { "type": "string" }
                  }
                }
              }
            }
          },
          "401": { "description": "Wrong secret" },
          "409": { "description": "Already initialized" }
        }
      }
    },
    "/register": {
      "post": {
        "tags": ["Users"],
        "summary": "Register a new user account",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["email", "password"],
                "properties": {
                  "email":    { "type": "string" },
                  "password": { "type": "string", "format": "password" }
                }
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Registered",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "id":    { "type": "string" },
                    "email": { "type": "string" },
                    "role":  { "type": "string", "example": "user" }
                  }
                }
              }
            }
          },
          "400": { "description": "Missing email or password" },
          "409": { "description": "Email already exists" }
        }
      }
    },
    "/oauth/authorize": {
      "get": {
        "tags": ["OAuth"],
        "summary": "Show login form",
        "parameters": [
          { "name": "client_id",    "in": "query", "required": true,  "schema": { "type": "string" } },
          { "name": "redirect_uri", "in": "query", "required": true,  "schema": { "type": "string" } },
          { "name": "state",        "in": "query", "required": false, "schema": { "type": "string" } },
          { "name": "scope",        "in": "query", "required": false, "schema": { "type": "string", "default": "openid" } }
        ],
        "responses": {
          "200": { "description": "HTML login form" }
        }
      },
      "post": {
        "tags": ["OAuth"],
        "summary": "Submit credentials and get authorization code",
        "requestBody": {
          "required": true,
          "content": {
            "application/x-www-form-urlencoded": {
              "schema": {
                "type": "object",
                "required": ["client_id", "redirect_uri", "email", "password"],
                "properties": {
                  "client_id":    { "type": "string" },
                  "redirect_uri": { "type": "string" },
                  "email":        { "type": "string" },
                  "password":     { "type": "string", "format": "password" },
                  "state":        { "type": "string" },
                  "scope":        { "type": "string" }
                }
              }
            }
          }
        },
        "responses": {
          "302": { "description": "Redirect to redirect_uri with code and state" },
          "400": { "description": "Invalid request" }
        }
      }
    },
    "/oauth/token": {
      "post": {
        "tags": ["OAuth"],
        "summary": "Exchange authorization code or refresh token",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "oneOf": [
                  {
                    "title": "Authorization Code Grant",
                    "type": "object",
                    "required": ["grant_type", "code", "redirect_uri", "client_id", "client_secret"],
                    "properties": {
                      "grant_type":    { "type": "string", "enum": ["authorization_code"] },
                      "code":          { "type": "string" },
                      "redirect_uri":  { "type": "string" },
                      "client_id":     { "type": "string" },
                      "client_secret": { "type": "string" }
                    }
                  },
                  {
                    "title": "Refresh Token Grant",
                    "type": "object",
                    "required": ["grant_type", "refresh_token", "client_id", "client_secret"],
                    "properties": {
                      "grant_type":    { "type": "string", "enum": ["refresh_token"] },
                      "refresh_token": { "type": "string" },
                      "client_id":     { "type": "string" },
                      "client_secret": { "type": "string" }
                    }
                  }
                ]
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Token issued",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/TokenResponse" }
              }
            }
          },
          "400": { "description": "Invalid grant type or missing fields" },
          "401": { "description": "Bad credentials or expired code" }
        }
      }
    },
    "/oauth/revoke": {
      "post": {
        "tags": ["OAuth"],
        "summary": "Revoke a refresh token (RFC 7009)",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["token"],
                "properties": {
                  "token": { "type": "string" }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "Always 200 (RFC 7009)" }
        }
      }
    },
    "/api/me": {
      "get": {
        "tags": ["Users"],
        "summary": "Get own profile from JWT",
        "security": [{ "bearerAuth": [] }],
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "id":    { "type": "string" },
                    "email": { "type": "string" },
                    "role":  { "type": "string" },
                    "scope": { "type": "string" }
                  }
                }
              }
            }
          },
          "401": { "description": "Invalid or missing token" }
        }
      }
    },
    "/api/users/{id}": {
      "get": {
        "tags": ["Users"],
        "summary": "Get user by ID (own profile or any if admin)",
        "security": [{ "bearerAuth": [] }],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/User" }
              }
            }
          },
          "401": { "description": "Invalid or missing token" },
          "403": { "description": "Requesting another user without admin role" },
          "404": { "description": "User not found" }
        }
      }
    },
    "/api/admin/users": {
      "get": {
        "tags": ["Users"],
        "summary": "List all users",
        "security": [{ "bearerAuth": [] }],
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "type": "array",
                  "items": { "$ref": "#/components/schemas/User" }
                }
              }
            }
          },
          "401": { "description": "Invalid or missing token" }
        }
      },
      "post": {
        "tags": ["Users"],
        "summary": "Create a new user",
        "security": [{ "bearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["email", "password"],
                "properties": {
                  "email":    { "type": "string" },
                  "password": { "type": "string", "format": "password" },
                  "role":     { "type": "string", "enum": ["user", "admin"], "default": "user" }
                }
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Created",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/User" }
              }
            }
          },
          "400": { "description": "Missing email or password" },
          "401": { "description": "Invalid or missing token" },
          "409": { "description": "Email already exists" }
        }
      }
    },
    "/api/notes": {
      "get": {
        "tags": ["Notes"],
        "summary": "List notes for authenticated user",
        "security": [{ "bearerAuth": [] }],
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "type": "array",
                  "items": { "$ref": "#/components/schemas/Note" }
                }
              }
            }
          },
          "401": { "description": "Invalid or missing token" }
        }
      },
      "post": {
        "tags": ["Notes"],
        "summary": "Create a note",
        "security": [{ "bearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["title"],
                "properties": {
                  "title":   { "type": "string" },
                  "content": { "type": "string" }
                }
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Created",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/Note" }
              }
            }
          },
          "400": { "description": "Title required" },
          "401": { "description": "Invalid or missing token" }
        }
      }
    },
    "/api/compile": {
      "post": {
        "tags": ["LaTeX"],
        "summary": "Compile LaTeX source to PDF (cached by MD5)",
        "description": "Computes MD5(engine+source). Returns a cached PDF from R2 on a hit (X-Cache: HIT), or calls the upstream compile server, stores the result, and returns it on a miss (X-Cache: MISS).",
        "security": [{ "bearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["source"],
                "properties": {
                  "source": { "type": "string", "description": "Full LaTeX source" },
                  "engine": { "type": "string", "enum": ["lualatex", "pdflatex", "xelatex"], "default": "lualatex" }
                }
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "PDF bytes",
            "headers": {
              "X-Cache": { "schema": { "type": "string", "enum": ["HIT", "MISS"] } }
            },
            "content": { "application/pdf": { "schema": { "type": "string", "format": "binary" } } }
          },
          "400": { "description": "Missing or empty source" },
          "401": { "description": "Invalid or missing token" },
          "422": { "description": "Compile error", "content": { "application/json": { "schema": { "type": "object", "properties": { "error": { "type": "string" }, "log": { "type": "string" } } } } } },
          "502": { "description": "Compile server unreachable" }
        }
      }
    },
    "/api/latex-files": {
      "get": {
        "tags": ["LaTeX"],
        "summary": "List LaTeX files for authenticated user",
        "security": [{ "bearerAuth": [] }],
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "type": "array",
                  "items": { "$ref": "#/components/schemas/LatexFile" }
                }
              }
            }
          },
          "401": { "description": "Invalid or missing token" }
        }
      },
      "post": {
        "tags": ["LaTeX"],
        "summary": "Create a new LaTeX file",
        "security": [{ "bearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["name"],
                "properties": {
                  "name":    { "type": "string" },
                  "content": { "type": "string", "default": "" },
                  "engine":  { "type": "string", "enum": ["lualatex", "pdflatex", "xelatex"], "default": "lualatex" }
                }
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Created",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/LatexFile" }
              }
            }
          },
          "400": { "description": "Missing name" },
          "401": { "description": "Invalid or missing token" }
        }
      }
    },
    "/api/latex-files/{id}": {
      "get": {
        "tags": ["LaTeX"],
        "summary": "Get a LaTeX file with content",
        "security": [{ "bearerAuth": [] }],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/LatexFileWithContent" }
              }
            }
          },
          "401": { "description": "Invalid or missing token" },
          "404": { "description": "Not found" }
        }
      },
      "put": {
        "tags": ["LaTeX"],
        "summary": "Update a LaTeX file (name, engine, and/or content)",
        "security": [{ "bearerAuth": [] }],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "name":    { "type": "string" },
                  "content": { "type": "string" },
                  "engine":  { "type": "string", "enum": ["lualatex", "pdflatex", "xelatex"] }
                }
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Updated",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/LatexFile" }
              }
            }
          },
          "400": { "description": "Invalid request body" },
          "401": { "description": "Invalid or missing token" },
          "404": { "description": "Not found" }
        }
      },
      "delete": {
        "tags": ["LaTeX"],
        "summary": "Delete a LaTeX file",
        "security": [{ "bearerAuth": [] }],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "204": { "description": "Deleted" },
          "401": { "description": "Invalid or missing token" },
          "404": { "description": "Not found" }
        }
      }
    },
    "/api/notes/{id}": {
      "get": {
        "tags": ["Notes"],
        "summary": "Get a note by ID",
        "security": [{ "bearerAuth": [] }],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/Note" }
              }
            }
          },
          "401": { "description": "Invalid or missing token" },
          "404": { "description": "Not found" }
        }
      },
      "delete": {
        "tags": ["Notes"],
        "summary": "Delete a note by ID",
        "security": [{ "bearerAuth": [] }],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "204": { "description": "Deleted" },
          "401": { "description": "Invalid or missing token" }
        }
      }
    }
  }
}`

const docsHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>sir-backend API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  <style>
    body { margin: 0; }
    #swagger-ui .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "/api/docs/openapi.json",
      dom_id: "#swagger-ui",
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: "BaseLayout",
      deepLinking: true,
      persistAuthorization: true,
    });
  </script>
</body>
</html>`

// DocsJSON serves the raw OpenAPI 3.0 spec.
func DocsJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(openapiJSON))
}

// DocsUI serves the Swagger UI HTML page.
func DocsUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(docsHTML))
}
