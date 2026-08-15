package app

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const radioReferenceEndpoint = "https://api.radioreference.com/soap2/index.php"

type RadioReferenceStatus struct {
	State    string `json:"state"`
	Endpoint string `json:"endpoint"`
	Note     string `json:"note"`
}

type ReferenceLocation struct {
	ZIP       int     `json:"zip"`
	City      string  `json:"city"`
	StateID   int     `json:"stateID"`
	CountyID  int     `json:"countyID"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type ReferenceCounty struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ReferenceChannel struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	FrequencyHz float64 `json:"frequencyHz"`
	InputHz     float64 `json:"inputHz"`
	Mode        string  `json:"mode"`
	Tone        string  `json:"tone"`
	County      string  `json:"county"`
}

type ReferenceP25System struct {
	ID                int                   `json:"id"`
	Name              string                `json:"name"`
	NAC               string                `json:"nac"`
	WACN              string                `json:"wacn"`
	SystemID          string                `json:"systemID"`
	ControlChannelsHz []float64             `json:"controlChannelsHz"`
	Talkgroups        []TalkgroupDefinition `json:"talkgroups"`
}

type RadioReferenceNearbyResult struct {
	Location    ReferenceLocation    `json:"location"`
	RadiusMiles float64              `json:"radiusMiles"`
	Counties    []ReferenceCounty    `json:"counties"`
	Channels    []ReferenceChannel   `json:"channels"`
	P25Systems  []ReferenceP25System `json:"p25Systems"`
}

type radioReferenceClient struct {
	username string
	password string
	appKey   string
	endpoint string
	client   *http.Client
}

type soapValue struct {
	name, value, valueType string
}

type xmlNode struct {
	XMLName  xml.Name
	Attrs    []xml.Attr
	Text     string
	Children []*xmlNode
}

func (node *xmlNode) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {
	node.XMLName, node.Attrs = start.Name, start.Attr
	for {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		switch value := token.(type) {
		case xml.StartElement:
			child := &xmlNode{}
			if err := decoder.DecodeElement(child, &value); err != nil {
				return err
			}
			node.Children = append(node.Children, child)
		case xml.CharData:
			node.Text += string(value)
		case xml.EndElement:
			return nil
		}
	}
}

func newRadioReferenceClient() *radioReferenceClient {
	endpoint := strings.TrimSpace(os.Getenv("GPSDR_RR_ENDPOINT"))
	if endpoint == "" {
		endpoint = radioReferenceEndpoint
	}
	return &radioReferenceClient{
		username: firstEnvironment("GPSDR_RR_USERNAME"),
		password: firstEnvironment("GPSDR_RR_PASSWORD"),
		appKey:   firstEnvironment("GPSDR_RR_APP_KEY"),
		endpoint: endpoint,
		client:   &http.Client{Timeout: 35 * time.Second},
	}
}

func (client *radioReferenceClient) Status() RadioReferenceStatus {
	if client.username == "" || client.password == "" || client.appKey == "" {
		return RadioReferenceStatus{State: "setup", Endpoint: client.endpoint,
			Note: "Add a premium account and approved API key through environment variables."}
	}
	return RadioReferenceStatus{State: "ready", Endpoint: client.endpoint, Note: "RadioReference location import is configured."}
}

func (client *radioReferenceClient) Nearby(parent context.Context, zipCode int, radiusMiles float64) (RadioReferenceNearbyResult, error) {
	if client.Status().State != "ready" {
		return RadioReferenceNearbyResult{}, errors.New("RadioReference credentials and an approved API key are required")
	}
	if zipCode < 1000 || zipCode > 99999 {
		return RadioReferenceNearbyResult{}, errors.New("enter a valid US ZIP code")
	}
	if radiusMiles < 1 || radiusMiles > 100 {
		return RadioReferenceNearbyResult{}, errors.New("range must be between 1 and 100 miles")
	}
	contextWithTimeout, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	zipReturn, err := client.call(contextWithTimeout, "getZipcodeInfo", soapValue{"zipcode", strconv.Itoa(zipCode), "xsd:int"})
	if err != nil {
		return RadioReferenceNearbyResult{}, err
	}
	location := ReferenceLocation{ZIP: zipCode, City: field(zipReturn, "city"), StateID: intField(zipReturn, "stid"),
		CountyID: intField(zipReturn, "ctid"), Latitude: floatField(zipReturn, "lat"), Longitude: floatField(zipReturn, "lon")}
	if location.StateID == 0 || location.CountyID == 0 {
		return RadioReferenceNearbyResult{}, errors.New("RadioReference did not return a county for that ZIP code")
	}
	counties, err := client.nearbyCounties(contextWithTimeout, location, radiusMiles)
	if err != nil {
		return RadioReferenceNearbyResult{}, err
	}
	result := RadioReferenceNearbyResult{Location: location, RadiusMiles: radiusMiles}
	for _, county := range counties {
		result.Counties = append(result.Counties, ReferenceCounty{ID: county.ID, Name: county.Name})
	}
	result.Channels, err = client.channelsForCounties(contextWithTimeout, counties)
	if err != nil {
		return RadioReferenceNearbyResult{}, err
	}
	result.P25Systems, err = client.p25ForCounties(contextWithTimeout, counties, location, radiusMiles)
	if err != nil {
		return RadioReferenceNearbyResult{}, err
	}
	return result, nil
}

type rrCountyInfo struct {
	ID            int
	Name          string
	Latitude      float64
	Longitude     float64
	Range         float64
	Subcategories []int
	Systems       map[int]string
}

func (client *radioReferenceClient) nearbyCounties(context context.Context, location ReferenceLocation, radius float64) ([]rrCountyInfo, error) {
	state, err := client.call(context, "getStateInfo", soapValue{"stid", strconv.Itoa(location.StateID), "xsd:int"})
	if err != nil {
		return nil, err
	}
	countyIDs := make([]int, 0)
	for _, item := range arrayUnder(state, "countyList") {
		if id := intField(item, "ctid"); id != 0 {
			countyIDs = append(countyIDs, id)
		}
	}
	if len(countyIDs) == 0 {
		countyIDs = append(countyIDs, location.CountyID)
	}
	type outcome struct {
		info rrCountyInfo
		err  error
	}
	jobs := make(chan int)
	results := make(chan outcome, len(countyIDs))
	var workers sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for id := range jobs {
				node, callErr := client.call(context, "getCountyInfo", soapValue{"ctid", strconv.Itoa(id), "xsd:int"})
				if callErr != nil {
					results <- outcome{err: callErr}
					continue
				}
				results <- outcome{info: parseCounty(node)}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, id := range countyIDs {
			select {
			case jobs <- id:
			case <-context.Done():
				return
			}
		}
	}()
	workers.Wait()
	close(results)
	counties := make([]rrCountyInfo, 0)
	for outcome := range results {
		if outcome.err != nil {
			continue
		}
		info := outcome.info
		distance := haversineMiles(location.Latitude, location.Longitude, info.Latitude, info.Longitude)
		if info.ID == location.CountyID || distance <= radius+math.Max(info.Range, 0) {
			counties = append(counties, info)
		}
	}
	if len(counties) == 0 {
		return nil, errors.New("no county data was returned inside the selected range")
	}
	sort.Slice(counties, func(i, j int) bool { return counties[i].Name < counties[j].Name })
	return counties, nil
}

func parseCounty(node *xmlNode) rrCountyInfo {
	info := rrCountyInfo{ID: intField(node, "ctid"), Name: field(node, "countyName"),
		Latitude: floatField(node, "lat"), Longitude: floatField(node, "lon"), Range: floatField(node, "range"), Systems: make(map[int]string)}
	for _, item := range arrayUnder(node, "cats") {
		for _, subcategory := range arrayUnder(item, "subcats") {
			if id := intField(subcategory, "scid"); id != 0 {
				info.Subcategories = append(info.Subcategories, id)
			}
		}
	}
	for _, item := range arrayUnder(node, "trsList") {
		if id := intField(item, "sid"); id != 0 {
			info.Systems[id] = field(item, "sName")
		}
	}
	return info
}

func (client *radioReferenceClient) channelsForCounties(context context.Context, counties []rrCountyInfo) ([]ReferenceChannel, error) {
	type job struct {
		subcategory int
		county      string
	}
	jobs := make([]job, 0)
	for _, county := range counties {
		for _, subcategory := range county.Subcategories {
			if len(jobs) >= 300 {
				break
			}
			jobs = append(jobs, job{subcategory, county.Name})
		}
	}
	jobQueue := make(chan job)
	results := make(chan []ReferenceChannel, len(jobs))
	var workers sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for item := range jobQueue {
				node, err := client.call(context, "getSubcatFreqs", soapValue{"scid", strconv.Itoa(item.subcategory), "xsd:int"})
				if err != nil {
					continue
				}
				batch := make([]ReferenceChannel, 0)
				for _, frequency := range arrayItems(node) {
					outMHz := floatField(frequency, "out")
					if outMHz <= 0 {
						continue
					}
					name := field(frequency, "alpha")
					if name == "" {
						name = field(frequency, "descr")
					}
					batch = append(batch, ReferenceChannel{ID: fmt.Sprintf("%d-%.6f", item.subcategory, outMHz), Name: name,
						Description: field(frequency, "descr"), FrequencyHz: outMHz * 1e6,
						InputHz: floatField(frequency, "in") * 1e6, Mode: normalizedMode(field(frequency, "mode")),
						Tone: field(frequency, "tone"), County: item.county})
				}
				select {
				case results <- batch:
				case <-context.Done():
					return
				}
			}
		}()
	}
	go func() {
		defer close(jobQueue)
		for _, item := range jobs {
			select {
			case jobQueue <- item:
			case <-context.Done():
				return
			}
		}
	}()
	go func() {
		workers.Wait()
		close(results)
	}()
	channels := make([]ReferenceChannel, 0)
	seen := make(map[string]bool)
	for batch := range results {
		for _, channel := range batch {
			if !seen[channel.ID] {
				seen[channel.ID] = true
				channels = append(channels, channel)
			}
		}
	}
	sort.Slice(channels, func(i, j int) bool { return channels[i].FrequencyHz < channels[j].FrequencyHz })
	return channels, nil
}

func (client *radioReferenceClient) p25ForCounties(context context.Context, counties []rrCountyInfo, location ReferenceLocation, radius float64) ([]ReferenceP25System, error) {
	systemNames := make(map[int]string)
	for _, county := range counties {
		for id, name := range county.Systems {
			systemNames[id] = name
		}
	}
	ids := make([]int, 0, len(systemNames))
	for id := range systemNames {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	if len(ids) > 40 {
		ids = ids[:40]
	}
	type p25Job struct {
		id   int
		name string
	}
	jobQueue := make(chan p25Job)
	results := make(chan ReferenceP25System, len(ids))
	var workers sync.WaitGroup
	for worker := 0; worker < 6; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for job := range jobQueue {
				if system, ok := client.p25System(context, job.id, job.name, location, radius); ok {
					select {
					case results <- system:
					case <-context.Done():
						return
					}
				}
			}
		}()
	}
	go func() {
		defer close(jobQueue)
		for _, id := range ids {
			select {
			case jobQueue <- p25Job{id: id, name: systemNames[id]}:
			case <-context.Done():
				return
			}
		}
	}()
	go func() {
		workers.Wait()
		close(results)
	}()
	systems := make([]ReferenceP25System, 0, len(ids))
	for system := range results {
		systems = append(systems, system)
	}
	sort.Slice(systems, func(i, j int) bool {
		if systems[i].Name == systems[j].Name {
			return systems[i].ID < systems[j].ID
		}
		return systems[i].Name < systems[j].Name
	})
	return systems, nil
}

func (client *radioReferenceClient) p25System(context context.Context, id int, name string, location ReferenceLocation, radius float64) (ReferenceP25System, bool) {
	details, err := client.call(context, "getTrsDetails", soapValue{"sid", strconv.Itoa(id), "xsd:int"})
	if err != nil {
		return ReferenceP25System{}, false
	}
	wacn, systemID := "", ""
	for _, sysid := range arrayUnder(details, "sysid") {
		if wacn == "" {
			wacn, systemID = field(sysid, "wacn"), field(sysid, "sysid")
		}
	}
	if wacn == "" && systemID == "" {
		return ReferenceP25System{}, false
	}
	sites, err := client.call(context, "getTrsSites", soapValue{"sid", strconv.Itoa(id), "xsd:int"})
	if err != nil {
		return ReferenceP25System{}, false
	}
	system := ReferenceP25System{ID: id, Name: name, WACN: wacn, SystemID: systemID}
	for _, site := range arrayItems(sites) {
		latitude, longitude := floatField(site, "lat"), floatField(site, "lon")
		if latitude != 0 && longitude != 0 && haversineMiles(location.Latitude, location.Longitude, latitude, longitude) > radius+floatField(site, "range") {
			continue
		}
		if system.NAC == "" {
			system.NAC = field(site, "nac")
		}
		for _, frequency := range arrayUnder(site, "siteFreqs") {
			mhz := floatField(frequency, "freq")
			usage := strings.ToLower(field(frequency, "use"))
			if mhz > 0 && (strings.Contains(usage, "c") || strings.Contains(usage, "control")) {
				system.ControlChannelsHz = appendUniqueFrequency(system.ControlChannelsHz, mhz*1e6)
			}
		}
	}
	if len(system.ControlChannelsHz) == 0 {
		return ReferenceP25System{}, false
	}
	talkgroups, err := client.call(context, "getTrsTalkgroups",
		soapValue{"sid", strconv.Itoa(id), "xsd:int"}, soapValue{"tgCid", "0", "xsd:int"},
		soapValue{"tgTag", "0", "xsd:int"}, soapValue{"tgDec", "0", "xsd:int"})
	if err == nil {
		for _, talkgroup := range arrayItems(talkgroups) {
			tgid := intField(talkgroup, "tgDec")
			if tgid == 0 {
				continue
			}
			name := field(talkgroup, "tgAlpha")
			if name == "" {
				name = field(talkgroup, "tgDescr")
			}
			encrypted := intField(talkgroup, "enc") != 0 || strings.EqualFold(field(talkgroup, "tgMode"), "E")
			system.Talkgroups = append(system.Talkgroups, TalkgroupDefinition{ID: tgid, Name: name,
				Mode: field(talkgroup, "tgMode"), Encrypted: encrypted, Enabled: !encrypted})
		}
	}
	return system, true
}

func (client *radioReferenceClient) call(context context.Context, method string, values ...soapValue) (*xmlNode, error) {
	if client.Status().State != "ready" {
		return nil, errors.New("RadioReference is not configured")
	}
	body := soapEnvelope(method, values, client)
	request, err := http.NewRequestWithContext(context, http.MethodPost, client.endpoint, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "text/xml; charset=utf-8")
	request.Header.Set("SOAPAction", "http://api.radioreference.com/soap2#"+method)
	response, err := client.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 16*1024*1024))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("RadioReference returned HTTP %d", response.StatusCode)
	}
	var root xmlNode
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if fault := findFirst(&root, "Fault"); fault != nil {
		message := field(fault, "faultstring")
		if message == "" {
			message = strings.TrimSpace(fault.Text)
		}
		return nil, errors.New("RadioReference: " + message)
	}
	result := findFirst(&root, "return")
	if result == nil {
		return nil, errors.New("RadioReference response had no return value")
	}
	index := make(map[string]*xmlNode)
	indexNodes(&root, index)
	result = resolveReference(result, index)
	resolveReferences(result, index, make(map[*xmlNode]bool))
	return result, nil
}

func soapEnvelope(method string, values []soapValue, client *radioReferenceClient) string {
	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	body.WriteString(`<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:tns="http://api.radioreference.com/soap2">`)
	body.WriteString("<soapenv:Body><tns:" + method + ` soapenv:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">`)
	for _, value := range values {
		body.WriteString("<" + value.name + ` xsi:type="` + value.valueType + `">` + xmlEscaped(value.value) + "</" + value.name + ">")
	}
	body.WriteString(`<authInfo xsi:type="tns:authInfo">`)
	for _, value := range []soapValue{{"username", client.username, "xsd:string"}, {"password", client.password, "xsd:string"},
		{"appKey", client.appKey, "xsd:string"}, {"version", Version, "xsd:string"}, {"style", "rpc", "xsd:string"}} {
		body.WriteString("<" + value.name + ` xsi:type="` + value.valueType + `">` + xmlEscaped(value.value) + "</" + value.name + ">")
	}
	body.WriteString("</authInfo></tns:" + method + "></soapenv:Body></soapenv:Envelope>")
	return body.String()
}

func xmlEscaped(value string) string {
	var buffer bytes.Buffer
	_ = xml.EscapeText(&buffer, []byte(value))
	return buffer.String()
}

func (node *xmlNode) attr(name string) string {
	for _, attribute := range node.Attrs {
		if attribute.Name.Local == name {
			return attribute.Value
		}
	}
	return ""
}

func indexNodes(node *xmlNode, index map[string]*xmlNode) {
	if id := strings.TrimPrefix(node.attr("id"), "#"); id != "" {
		index[id] = node
	}
	for _, child := range node.Children {
		indexNodes(child, index)
	}
}

func resolveReference(node *xmlNode, index map[string]*xmlNode) *xmlNode {
	if href := strings.TrimPrefix(node.attr("href"), "#"); href != "" {
		if resolved := index[href]; resolved != nil {
			return resolved
		}
	}
	return node
}

func resolveReferences(node *xmlNode, index map[string]*xmlNode, visited map[*xmlNode]bool) {
	if node == nil || visited[node] {
		return
	}
	visited[node] = true
	for childIndex, child := range node.Children {
		resolved := resolveReference(child, index)
		node.Children[childIndex] = resolved
		resolveReferences(resolved, index, visited)
	}
}

func findFirst(node *xmlNode, name string) *xmlNode {
	if strings.EqualFold(node.XMLName.Local, name) {
		return node
	}
	for _, child := range node.Children {
		if result := findFirst(child, name); result != nil {
			return result
		}
	}
	return nil
}

func directChild(node *xmlNode, name string) *xmlNode {
	for _, child := range node.Children {
		if strings.EqualFold(child.XMLName.Local, name) {
			return child
		}
	}
	return nil
}

func field(node *xmlNode, name string) string {
	if child := directChild(node, name); child != nil {
		return strings.TrimSpace(child.Text)
	}
	return ""
}

func intField(node *xmlNode, name string) int {
	value, _ := strconv.Atoi(field(node, name))
	return value
}

func floatField(node *xmlNode, name string) float64 {
	value, _ := strconv.ParseFloat(field(node, name), 64)
	return value
}

func arrayUnder(node *xmlNode, name string) []*xmlNode {
	container := directChild(node, name)
	if container == nil {
		container = findFirst(node, name)
	}
	if container == nil {
		return nil
	}
	return arrayItems(container)
}

func arrayItems(node *xmlNode) []*xmlNode {
	items := make([]*xmlNode, 0)
	for _, child := range node.Children {
		if strings.EqualFold(child.XMLName.Local, "item") {
			items = append(items, child)
		}
	}
	if len(items) == 0 && node.XMLName.Local == "item" {
		items = append(items, node)
	}
	return items
}

func normalizedMode(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	switch {
	case strings.Contains(value, "AM"):
		return "am"
	case strings.Contains(value, "WFM") || value == "FM":
		return "wfm"
	case strings.Contains(value, "P25") || strings.Contains(value, "DMR") || strings.Contains(value, "NXDN"):
		return "digital"
	default:
		return "nfm"
	}
}

func appendUniqueFrequency(values []float64, frequency float64) []float64 {
	for _, value := range values {
		if math.Abs(value-frequency) < 1 {
			return values
		}
	}
	return append(values, frequency)
}

func haversineMiles(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMiles = 3958.7613
	toRadians := math.Pi / 180
	dLat, dLon := (lat2-lat1)*toRadians, (lon2-lon1)*toRadians
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*toRadians)*math.Cos(lat2*toRadians)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusMiles * math.Asin(math.Sqrt(a))
}
