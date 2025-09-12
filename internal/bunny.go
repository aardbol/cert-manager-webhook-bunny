package internal

type ZoneResponse struct {
	Items        []Items `json:"Items"`
	CurrentPage  int     `json:"CurrentPage"`
	TotalItems   int     `json:"TotalItems"`
	HasMoreItems bool    `json:"HasMoreItems"`
}

type Items struct {
	Id                            int      `json:"Id"`
	Domain                        string   `json:"Domain"`
	Records                       []Record `json:"Records"`
	DateModified                  string   `json:"DateModified"`
	DateCreated                   string   `json:"DateCreated"`
	NameserversDetected           bool     `json:"NameserversDetected"`
	CustomNameserversEnabled      bool     `json:"CustomNameserversEnabled"`
	Nameserver1                   string   `json:"Nameserver1"`
	Nameserver2                   string   `json:"Nameserver2"`
	SoaEmail                      string   `json:"SoaEmail"`
	NameserversNextCheck          string   `json:"NameserversNextCheck"`
	LoggingEnabled                bool     `json:"LoggingEnabled"`
	LoggingIPAnonymizationEnabled bool     `json:"LoggingIPAnonymizationEnabled"`
	LogAnonymizationType          int      `json:"LogAnonymizationType"`
}

type Record struct {
	Id                    int                     `json:"Id"`
	Type                  int                     `json:"Type"`
	Ttl                   int                     `json:"Ttl"`
	Value                 string                  `json:"Value"`
	Name                  string                  `json:"Name"`
	Weight                int                     `json:"Weight"`
	Priority              int                     `json:"Priority"`
	Port                  int                     `json:"Port"`
	Flags                 int                     `json:"Flags"`
	Tag                   string                  `json:"Tag"`
	Accelerated           bool                    `json:"Accelerated"`
	AcceleratedPullZoneId int                     `json:"AcceleratedPullZoneId"`
	LinkName              string                  `json:"LinkName"`
	IPGeoLocationInfo     IPGeoLocationInfo       `json:"IPGeoLocationInfo"`
	GeolocationInfo       string                  `json:"GeolocationInfo"`
	MonitorStatus         int                     `json:"MonitorStatus"`
	MonitorType           int                     `json:"MonitorType"`
	GeolocationLatitude   float32                 `json:"GeolocationLatitude"`
	GeolocationLongitude  float32                 `json:"GeolocationLongitude"`
	EnviromentalVariables []EnviromentalVariables `json:"EnviromentalVariables"`
	LatencyZone           string                  `json:"LatencyZone"`
	SmartRoutingType      int                     `json:"SmartRoutingType"`
	Disabled              bool                    `json:"Disabled"`
	Comment               string                  `json:"Comment"`
}

type IPGeoLocationInfo struct {
	CountryCode      string `json:"CountryCode"`
	Country          string `json:"Country"`
	ASN              int    `json:"ASN"`
	OrganizationName string `json:"OrganizationName"`
	City             string `json:"City"`
}

type EnviromentalVariables struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}
