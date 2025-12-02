package handler

import "github.com/crossplane/function-sdk-go/resource"

type Handler interface {
	GetNamespaceID(des *resource.DesiredComposed) (int, error)
	GetPath(des *resource.DesiredComposed) (string, error)
	Exists(obs resource.ObservedComposed) bool
}
