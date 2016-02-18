package services

import "net/http"

type AppFileSystem interface {
	http.FileSystem
}
