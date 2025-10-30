package main

type Resource struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Status     map[string]interface{} `json:"status"`
	Spec       map[string]interface{} `json:"spec"`
}

type Status struct {
	Conditions []map[string]interface{} `json:"conditions,omitempty"`
}

type Condition struct {
	Message string `json:"message,omitempty"`
}

type Spec struct {
	DeletionPolicy string                 `json:"deletionPolicy"`
	ForProvider    map[string]interface{} `json:"forProvider"`
}

type ForProvider struct {
	Description string `json:"description,omitempty"`
	Name        string `json:"name,omitempty"`
	Path        string `json:"path"`
	NamespaceId int    `json:"namespaceId,omitempty"`
	ParentId    int    `json:"parentId,omitempty"`
}
