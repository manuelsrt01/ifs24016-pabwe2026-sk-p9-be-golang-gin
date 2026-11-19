package docs

import (
	"fmt"
	"strings"

	"github.com/swaggo/swag"
)

const docTemplate = `{
  "swagger": "2.0",
  "info": {
    "description": "Dokumentasi REST API Delcom Blog — Platform Publikasi Artikel Multi-Tier dengan AI Content Generator & Integrasi Search Engine",
    "title": "Delcom Blog REST API",
    "version": "1.0.0"
  },
  "host": "%s",
  "basePath": "/",
  "schemes": ["%s"],
  "securityDefinitions": {
    "BearerAuth": {
      "type": "apiKey",
      "name": "Authorization",
      "in": "header",
      "description": "Masukkan Bearer token JWT dengan format: Bearer <token>"
    }
  },
  "paths": {
    "/api/public/categories": {
      "get": {
        "tags": ["Public"],
        "summary": "Mengambil daftar seluruh kategori artikel blog",
        "produces": ["application/json"],
        "responses": {
          "200": {"description": "Daftar kategori artikel"}
        }
      }
    },
    "/api/public/posts": {
      "get": {
        "tags": ["Public"],
        "summary": "Mengambil daftar seluruh artikel terbit dengan pencarian dan filter kategori",
        "produces": ["application/json"],
        "parameters": [
          {"name": "q", "in": "query", "type": "string", "required": false, "description": "Kata kunci pencarian"},
          {"name": "category", "in": "query", "type": "string", "required": false, "description": "Filter kategori"}
        ],
        "responses": {
          "200": {"description": "Daftar artikel publik"}
        }
      }
    },
    "/api/public/posts/{slug}": {
      "get": {
        "tags": ["Public"],
        "summary": "Mengambil detail artikel publik terbit berdasarkan slug",
        "produces": ["application/json"],
        "parameters": [
          {"name": "slug", "in": "path", "type": "string", "required": true, "description": "Slug artikel"}
        ],
        "responses": {
          "200": {"description": "Detail artikel publik"},
          "404": {"description": "Artikel tidak ditemukan"}
        }
      }
    },
    "/api/auth/register": {
      "post": {
        "tags": ["Auth"],
        "summary": "Pendaftaran akun penulis (author) baru",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [
          {
            "name": "request",
            "in": "body",
            "required": true,
            "schema": {
              "type": "object",
              "required": ["username", "email", "password"],
              "properties": {
                "username": {"type": "string", "example": "penulis_baru"},
                "email": {"type": "string", "example": "penulis@delcom.org"},
                "password": {"type": "string", "example": "rahasia123"}
              }
            }
          }
        ],
        "responses": {
          "201": {"description": "Registrasi berhasil"},
          "400": {"description": "Input tidak valid"},
          "409": {"description": "Username atau email sudah terdaftar"}
        }
      }
    },
    "/api/auth/login": {
      "post": {
        "tags": ["Auth"],
        "summary": "Autentikasi kredensial pengguna dan penerbitan token JWT",
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [
          {
            "name": "request",
            "in": "body",
            "required": true,
            "schema": {
              "type": "object",
              "required": ["username", "password"],
              "properties": {
                "username": {"type": "string", "example": "member"},
                "password": {"type": "string", "example": "member"}
              }
            }
          }
        ],
        "responses": {
          "200": {"description": "Login berhasil"},
          "401": {"description": "Kredensial tidak valid"}
        }
      }
    },
    "/api/auth/logout": {
      "post": {
        "tags": ["Auth"],
        "summary": "Logout sesi pengguna",
        "security": [{"BearerAuth": []}],
        "produces": ["application/json"],
        "responses": {
          "200": {"description": "Logout berhasil"}
        }
      }
    },
    "/api/auth/profile": {
      "get": {
        "tags": ["Auth"],
        "summary": "Mengambil data profil penulis yang sedang login",
        "security": [{"BearerAuth": []}],
        "produces": ["application/json"],
        "responses": {
          "200": {"description": "Profil pengguna"}
        }
      },
      "put": {
        "tags": ["Auth"],
        "summary": "Memperbarui informasi profil penulis yang sedang login",
        "security": [{"BearerAuth": []}],
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [
          {
            "name": "request",
            "in": "body",
            "required": true,
            "schema": {
              "type": "object",
              "properties": {
                "username": {"type": "string"},
                "email": {"type": "string"},
                "password": {"type": "string"}
              }
            }
          }
        ],
        "responses": {
          "200": {"description": "Profil berhasil diperbarui"}
        }
      }
    },
    "/api/posts": {
      "get": {
        "tags": ["Posts"],
        "summary": "Mengambil seluruh daftar artikel milik penulis yang sedang login",
        "security": [{"BearerAuth": []}],
        "produces": ["application/json"],
        "responses": {
          "200": {"description": "Daftar artikel penulis"}
        }
      },
      "post": {
        "tags": ["Posts"],
        "summary": "Membuat artikel baru",
        "security": [{"BearerAuth": []}],
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [
          {
            "name": "request",
            "in": "body",
            "required": true,
            "schema": {
              "type": "object",
              "required": ["title", "content"],
              "properties": {
                "title": {"type": "string"},
                "category": {"type": "string", "example": "Teknologi"},
                "excerpt": {"type": "string"},
                "content": {"type": "string"},
                "cover_image_path": {"type": "string"},
                "is_published": {"type": "boolean"}
              }
            }
          }
        ],
        "responses": {
          "201": {"description": "Artikel berhasil dibuat"}
        }
      }
    },
    "/api/posts/stats": {
      "get": {
        "tags": ["Posts"],
        "summary": "Mengambil ringkasan 3 kartu statistik penulis",
        "security": [{"BearerAuth": []}],
        "produces": ["application/json"],
        "responses": {
          "200": {"description": "Statistik artikel penulis"}
        }
      }
    },
    "/api/posts/{id}": {
      "get": {
        "tags": ["Posts"],
        "summary": "Mengambil detail satu artikel milik penulis berdasarkan ID",
        "security": [{"BearerAuth": []}],
        "produces": ["application/json"],
        "parameters": [
          {"name": "id", "in": "path", "type": "integer", "required": true}
        ],
        "responses": {
          "200": {"description": "Detail artikel"}
        }
      },
      "put": {
        "tags": ["Posts"],
        "summary": "Memperbarui artikel milik penulis",
        "security": [{"BearerAuth": []}],
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [
          {"name": "id", "in": "path", "type": "integer", "required": true},
          {
            "name": "request",
            "in": "body",
            "required": true,
            "schema": {
              "type": "object",
              "properties": {
                "title": {"type": "string"},
                "category": {"type": "string"},
                "excerpt": {"type": "string"},
                "content": {"type": "string"},
                "cover_image_path": {"type": "string"},
                "is_published": {"type": "boolean"}
              }
            }
          }
        ],
        "responses": {
          "200": {"description": "Artikel diperbarui"}
        }
      },
      "delete": {
        "tags": ["Posts"],
        "summary": "Menghapus artikel milik penulis",
        "security": [{"BearerAuth": []}],
        "produces": ["application/json"],
        "parameters": [
          {"name": "id", "in": "path", "type": "integer", "required": true}
        ],
        "responses": {
          "200": {"description": "Artikel dihapus"}
        }
      }
    },
    "/api/posts/{id}/toggle-publish": {
      "patch": {
        "tags": ["Posts"],
        "summary": "Mengubah status publikasi artikel secara instan (Draf <-> Terbit)",
        "security": [{"BearerAuth": []}],
        "produces": ["application/json"],
        "parameters": [
          {"name": "id", "in": "path", "type": "integer", "required": true}
        ],
        "responses": {
          "200": {"description": "Status publikasi diubah"}
        }
      }
    },
    "/api/posts/ai-generate": {
      "post": {
        "tags": ["AI"],
        "summary": "Menghasilkan draf artikel utuh menggunakan AI yang terintegrasi dengan Search Engine",
        "security": [{"BearerAuth": []}],
        "consumes": ["application/json"],
        "produces": ["application/json"],
        "parameters": [
          {
            "name": "request",
            "in": "body",
            "required": true,
            "schema": {
              "type": "object",
              "required": ["topic"],
              "properties": {
                "topic": {"type": "string", "example": "Arsitektur Microservices dengan Golang Gin"},
                "category": {"type": "string", "example": "Teknologi"},
                "include_web_search": {"type": "boolean", "example": true}
              }
            }
          }
        ],
        "responses": {
          "200": {"description": "Draf artikel AI dan daftar sumber search engine"}
        }
      }
    },
    "/api/admin/users": {
      "get": {
        "tags": ["Admin"],
        "summary": "Mengambil data seluruh pengguna sistem beserta total artikel yang ditulis",
        "security": [{"BearerAuth": []}],
        "produces": ["application/json"],
        "responses": {
          "200": {"description": "Daftar seluruh pengguna"}
        }
      }
    },
    "/api/admin/posts": {
      "get": {
        "tags": ["Admin"],
        "summary": "Monitoring arsip seluruh artikel dari semua penulis dalam sistem",
        "security": [{"BearerAuth": []}],
        "produces": ["application/json"],
        "responses": {
          "200": {"description": "Arsip seluruh artikel"}
        }
      }
    },
    "/api/admin/users/{id}/posts": {
      "get": {
        "tags": ["Admin"],
        "summary": "Mengambil seluruh artikel yang ditulis oleh pengguna tertentu",
        "security": [{"BearerAuth": []}],
        "produces": ["application/json"],
        "parameters": [
          {"name": "id", "in": "path", "type": "integer", "required": true}
        ],
        "responses": {
          "200": {"description": "Daftar artikel pengguna"}
        }
      }
    }
  }
}`

var SwaggerInfo = &swag.Spec{
	Version:          "1.0.0",
	Host:             "localhost:8081",
	BasePath:         "/",
	Schemes:          []string{"http"},
	Title:            "Delcom Blog REST API",
	Description:      "Dokumentasi REST API Delcom Blog",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func GetSwaggerJSON(host, scheme string) string {
	h := strings.TrimSpace(host)
	if h == "" {
		h = "localhost:8081"
	}
	s := strings.TrimSpace(scheme)
	if s == "" {
		s = "http"
	}
	return fmt.Sprintf(docTemplate, h, s)
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
