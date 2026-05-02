package internal

type Zone struct {
	Domain  string   `json:"Domain"`
	Records []Record `json:"Records"`
}

type Record struct {
	Id    int    `json:"Id"`
	Type  int    `json:"Type"`
	Value string `json:"Value"`
	Name  string `json:"Name"`
}

// CreateRecordRequest is the request body for creating a Bunny DNS record.
type CreateRecordRequest struct {
	Type  int    `json:"Type"`
	Ttl   int    `json:"Ttl"`
	Value string `json:"Value"`
	Name  string `json:"Name"`
}
