package internal

type Zone struct {
	Records []Record `json:"Records"`
}

type Record struct {
	Id    int    `json:"Id"`
	Type  int    `json:"Type"`
	Value string `json:"Value"`
	Name  string `json:"Name"`
}
