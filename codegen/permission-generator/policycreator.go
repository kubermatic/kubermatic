package main

type PolicyCreator interface {
	GeneratePolicy(FuncInvocations) ([]byte, error)
}
