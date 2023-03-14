package main

func main() {
	parseFlags() // TODO
}

func parseFlags() {

}

func policyForPackages(poc PolicyCreator, pkgsToSearch []string, filter string) ([]byte, error) {
	invoc, err := SearchFuncInvocationsForPackages(pkgsToSearch, filter)
	if err != nil {
		return nil, err
	}
	return poc.GeneratePolicy(invoc)
}
