package handler


type InterfaceType interface{}

type SimpleAlias string

type SimpleStructure struct {
	Id   int
	Name string
}

type SimpleStructureWithAnnotations struct {
	Id   int    `json:"id"`
	Name string `json:"required,omitempty"`
}

type StructureWithSlice struct {
	Id   int
	Name []byte
}

type StructureWithEmbededStructure struct {
	StructureWithSlice
}
type StructureWithEmbededPointer struct {
	*StructureWithSlice
}

type APIError struct {
	ErrorCode    int
	ErrorMessage string
}
