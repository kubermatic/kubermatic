package api

type Command struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}
