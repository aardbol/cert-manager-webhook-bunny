package internal

type RecordType int

const (
	RecordTypeTXT RecordType = 3
)

type ZoneList struct {
	Items []Zone `json:"Items"`
}

type Zone struct {
	ID      int      `json:"Id"`
	Domain  string   `json:"Domain"`
	Records []Record `json:"Records"`
}

type Record struct {
	ID    int        `json:"Id"`
	Type  RecordType `json:"Type"`
	Value string     `json:"Value"`
	Name  string     `json:"Name"`
}

type CreateRecordRequest struct {
	Type  RecordType `json:"Type"`
	TTL   int        `json:"Ttl"`
	Value string     `json:"Value"`
	Name  string     `json:"Name"`
}
