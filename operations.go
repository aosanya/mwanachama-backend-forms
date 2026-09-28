package mwanachamaforms

import _ "embed"

//go:embed forms.operations.json
var operationsJSON []byte

func Operations() []byte { return operationsJSON }
