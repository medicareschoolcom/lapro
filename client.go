package lapro

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrNoResults = errors.New("no results")
)

const (
	// DefaultAPIBaseURL is the base URL of the production LAPro API.
	DefaultAPIBaseURL = "https://api.leadadvantagepro.com"
	// DefaultAuthBaseURL is the base URL of the production LAPro authorization server.
	DefaultAuthBaseURL = "https://authorize.leadadvantagepro.com"
)

const (
	GenderMale   = "M"
	GenderFemale = "F"
)

// Boolean is a custom type assisting in the proper unmarshaling of non-standard boolean types sent over json.
type Boolean bool

func (bit *Boolean) UnmarshalJSON(data []byte) error {
	asString := string(data)
	asString = strings.ToLower(asString)
	asString = strings.TrimPrefix(asString, `"`)
	asString = strings.TrimSuffix(asString, `"`)

	switch asString {
	case "true", "yes", "1":
		*bit = true
	case "false", "no", "0", "":
		*bit = false
	default:
		return fmt.Errorf("Boolean unmarshal error: invalid input %s", asString)
	}

	return nil
}

// Cent is a custom type for unmarshaling currency strings into int32 cents.
type Cent int32

func (c *Cent) UnmarshalJSON(data []byte) error {
	asString := string(data)
	asString = strings.TrimPrefix(asString, `"`)
	asString = strings.TrimSuffix(asString, `"`)

	// Handle empty or null values
	if asString == "" || asString == "null" {
		*c = 0
		return nil
	}

	// Parse as float to handle decimal currency values
	dollars, err := strconv.ParseFloat(asString, 64)
	if err != nil {
		return fmt.Errorf("Cent unmarshal error: invalid currency format %s", asString)
	}

	// Round negative dollars to 0
	if dollars < 0 {
		dollars = 0
	}

	// Convert to cents (multiply by 100 and round)
	cents := int32(dollars * 100)
	*c = Cent(cents)

	return nil
}

type County struct {
	Name string `json:"county_name"`
	Fips string `json:"county_fips"`
}

type Drug struct {
	DosageID             string  `json:"dosageid"`
	PackType             string  `json:"packtype"`
	Unit                 string  `json:"unit"`
	NDCCode              string  `json:"ndccode"`
	CommonMetricQuantity string  `json:"commonmetricquantity"`
	Strength             string  `json:"strength"`
	TradeName            string  `json:"tradename"`
	DrugName             string  `json:"drugname"`
	DrugID               string  `json:"drugid"`
	IsCommonDosage       Boolean `json:"iscommondosage"`
	DrugType             string  `json:"drugtype"`
	GenericDrugName      string  `json:"genericdrugname"`
	PackSize             string  `json:"packsize"`
	GenericDosageID      string  `json:"genericdosageid"`
	GenericDrugID        string  `json:"genericdrugid"`
	CommonDaysOfSupply   string  `json:"commondaysofsupply"`
	CommonUserQuantity   string  `json:"commonuserquantity"`
}

type Pharmacy struct {
	PharmacyID string `json:"pharmacyid"`
	Name       string `json:"name"`
	Distance   string `json:"distance"`
	Address1   string `json:"address1"`
	Address2   string `json:"address2"`
	City       string `json:"city"`
	State      string `json:"state"`
	Zip        string `json:"zip"`
}

type ListCountiesByZipCodeResponse struct {
	StateID   string   `json:"state_id"`
	StateName string   `json:"state_name"`
	Counties  []County `json:"counties"`
}

type RequestQuoteParams struct {
	Zip               string             `json:"zip"`
	County            string             `json:"county"`
	DOB               string             `json:"dob"`
	EffectiveDate     string             `json:"eff_date"`
	HealthStatus      string             `json:"health_status"`
	QuoteProducts     QuoteProducts      `json:"quote_products"`
	ExternalProviders []ExternalProvider `json:"external_providers"`
	Drugs             []QuoteDrug        `json:"drugs"`
	Gender            string             `json:"gender"`
	PharmacyID        string             `json:"pharmacy_id"`
}

type QuoteProducts struct {
	PartD             bool `json:"partd"`
	MedicareAdvantage bool `json:"ma"`
	MAPD              bool `json:"mapd"`
}

type QuoteDrug struct {
	Name             string `json:"drug_name"`
	Type             string `json:"drug_type"`
	MetricQty        string `json:"drug_metric_qty"`
	Freq             string `json:"drug_freq"`
	NDCCode          string `json:"drug_ndc_code"`
	Strength         string `json:"drug_strength"`
	PackSize         string `json:"drug_pack_size"`
	PackType         string `json:"drug_pack_type"`
	PackUnit         string `json:"drug_pack_unit"`
	DosageExternalID string `json:"drug_dosage_external_id"`
}

type ExternalProvider struct {
	ExternalProviderTypeID     string `json:"external_provider_type_id"`
	ExternalProviderID         string `json:"external_provider_id"`
	ExternalProviderLocationID string `json:"external_provider_location_id"`
}

type RequestQuoteResponse struct {
	MapDResults []*Quote `json:"mapdresult"`
}

type Quote struct {
	PlanID                       string `json:"plan_id"`
	PlanName                     string `json:"plan_name"`
	CarrierName                  string `json:"carrier_name"`
	CarrierID                    string `json:"carrier_id"`
	DrugPremiumCents             Cent   `json:"drug_premium"`
	EstimatedAnnualDrugCostCents Cent   `json:"estimated_annual_drug_cost"`
	MAPDProductType              string `json:"mapd_product_type"`
	AnnualTotalCombinedCents     Cent   `json:"annual_total_combined"`
}

func (q *Quote) EstimatedMonthlyDrugCostCents() Cent {
	return q.EstimatedAnnualDrugCostCents / 12
}

type Carrier struct {
	CarrierID string `json:"carrier_id"`
	CompanyID string `json:"company_id"`
	ShortName string `json:"short_name"`
	Name      string `json:"name"`
	Active    string `json:"active"`
	NAIC      string `json:"naic"`
}

type GetDrugNamesResponse struct {
	DrugName string `json:"drug_name"`
	DrugID   string `json:"drug_id"`
}

// Error represents an error response from the LAPro API. The structure varies based on the type of error:
// For successful API calls that return no results (e.g., valid searches with zero records), only the
// Result, Message, and Success fields are populated. For actual API errors, all fields are populated
// with the status code indicating the issue type and the complete error response details.
type Error struct {
	Message      string          `json:"message"`
	Success      string          `json:"success"`
	StatusCode   string          `json:"statuscode"`
	HTTPMethod   string          `json:"httpmethod"`
	HTTPContent  json.RawMessage `json:"httpcontent"`
	CurrentEvent string          `json:"currentevent"`
}

func (e Error) Error() string {
	return fmt.Sprintf("lapro error: message=%s success=%s statuscode=%s httpmethod=%s httpcontent=%s currentevent=%s",
		e.Message, e.Success, e.StatusCode, e.HTTPMethod, string(e.HTTPContent), e.CurrentEvent)
}

type requestAccessTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in,string"`
	TokenType    string `json:"token_type"`
	UserID       int64  `json:"user_id"`
	RefreshToken string `json:"refresh_token"`
	Context      string `json:"context"`
}

type requestAccessTokenWithRefreshResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in,string"`
	TokenType    string `json:"token_type"`
	UserID       int64  `json:"user_id"`
	RefreshToken string `json:"refresh_token"`
}

type Client struct {
	apiBaseURL   string
	authBaseURL  string
	username     string
	password     string
	clientID     string
	clientSecret string
	httpClient   *http.Client

	// Access token cols that are protected from mutation
	// by the mutex. Do no directly mutate.
	mu                sync.Mutex
	accessToken       string
	refreshToken      string
	accessTokenExpiry time.Time
}

// NewClient returns a new LAPro client. Use DefaultAPIBaseURL and DefaultAuthBaseURL
// to target the production LAPro environment.
func NewClient(apiBaseURL, authBaseURL, username, password, clientID, clientSecret string, httpClient *http.Client) *Client {
	return &Client{
		apiBaseURL:   strings.TrimSuffix(apiBaseURL, "/"),
		authBaseURL:  strings.TrimSuffix(authBaseURL, "/"),
		username:     username,
		password:     password,
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   httpClient,
	}
}

// Ping checks the current health status of the client. A client with an active access token
// is in a healthy state.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.getAccessToken(ctx)
	return err
}

// ListCountiesByZipCode retrieves all counties within a given zip code, organized by state.
// The response includes one entry per state that intersects with the zip code, with each
// state entry containing its respective counties that fall within the zip code boundaries.
func (c *Client) ListCountiesByZipCode(ctx context.Context, zipCode string) ([]*ListCountiesByZipCodeResponse, error) {
	url := c.apiBaseURL + "/api/geo/countiesByZip/" + zipCode
	const method = "GET"

	accessToken, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new request: %w", err)
	}

	req.Header.Add("Authorization", accessToken)
	req = req.WithContext(ctx)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute http request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	switch res.StatusCode {
	case http.StatusOK:
		var out []*ListCountiesByZipCodeResponse
		if err = json.Unmarshal(body, &out); err != nil {

			// If the json unmarshalling has failed, it could mean that the api has returned
			// an error message.
			var laproErr Error
			if err := json.Unmarshal(body, &laproErr); err != nil {
				return nil, fmt.Errorf("failed to parse response body as json: %s", string(body))
			}

			if laproErr.Success == "0" {
				return nil, ErrNoResults
			}

			return nil, laproErr
		}
		return out, nil
	default:
		return nil, fmt.Errorf("bad status code: %s", string(body))
	}
}

// GetDrugNames returns a list of drug names that match the provided partial drug name. A partial
// drug name typically consist of the first 3 letters of the drug name.
func (c *Client) GetDrugNames(ctx context.Context, searchTerm string) ([]*GetDrugNamesResponse, error) {
	url := c.apiBaseURL + "/api/drug/getDrugNames"
	const method = "POST"

	accessToken, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	payload, err := json.Marshal(map[string]string{"drug_name": searchTerm})
	if err != nil {
		return nil, fmt.Errorf("failed to json serialize payload: %w", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create new request: %w", err)
	}

	req.Header.Add("Authorization", accessToken)
	req = req.WithContext(ctx)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute http request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	switch res.StatusCode {
	case http.StatusOK:
		var out []*GetDrugNamesResponse
		if err = json.Unmarshal(body, &out); err != nil {
			// If the json unmarshalling has failed, it could mean that the api has returned
			// an error message.
			var laproErr Error
			if err := json.Unmarshal(body, &laproErr); err != nil {
				return nil, fmt.Errorf("failed to parse response body as json: %s", string(body))
			}

			if laproErr.Success == "0" {
				return []*GetDrugNamesResponse{}, nil
			}

			return nil, laproErr
		}
		return out, nil
	default:
		return nil, errors.New(string(body))
	}
}

// ListDrugDosagesByDrugName returns a list of Drugs & dosages for a given drug name.
func (c *Client) ListDrugDosagesByDrugName(ctx context.Context, drugName string) ([]*Drug, error) {
	url := c.apiBaseURL + "/api/drug/searchByDrugName"
	const method = "POST"

	accessToken, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	payload, err := json.Marshal(map[string]string{"drug_name": drugName})
	if err != nil {
		return nil, fmt.Errorf("failed to json serialize payload: %w", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create new request: %w", err)
	}

	req.Header.Add("Authorization", accessToken)
	req = req.WithContext(ctx)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute http request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	switch res.StatusCode {
	case http.StatusOK:
		var drugs []*Drug
		if err = json.Unmarshal(body, &drugs); err != nil {
			// If the json unmarshalling has failed, it could mean that the api has returned
			// an error message.
			var laproErr Error
			if err := json.Unmarshal(body, &laproErr); err != nil {
				return nil, fmt.Errorf("failed to parse response body as json: %s", string(body))
			}

			if laproErr.Success == "0" {
				return []*Drug{}, nil
			}

			return nil, laproErr
		}
		return drugs, nil
	default:
		return nil, errors.New(string(body))
	}
}

// ListPharmaciesByZipCode returns a list of pharmacies for a given zipcode, within a given radius from the
// geographical center of the zipcode.
func (c *Client) ListPharmaciesByZipCode(ctx context.Context, zipCode, radius string) ([]*Pharmacy, error) {
	url := c.apiBaseURL + "/api/pharmacy/searchByZip"
	const method = "POST"

	accessToken, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	payload, err := json.Marshal(map[string]string{
		"zip":    zipCode,
		"radius": radius,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to json serialize payload: %w", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create new request: %w", err)
	}

	req.Header.Add("Authorization", accessToken)
	req = req.WithContext(ctx)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute http request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	switch res.StatusCode {
	case http.StatusOK:
		var pharmacies []*Pharmacy
		if err = json.Unmarshal(body, &pharmacies); err != nil {
			// If the json unmarshalling has failed, it could mean that the api has returned
			// an error message.
			var laproErr Error
			if err := json.Unmarshal(body, &laproErr); err != nil {
				return nil, fmt.Errorf("failed to parse response body as json: %s", string(body))
			}

			if laproErr.Success == "0" {
				return []*Pharmacy{}, nil
			}

			// Bad status
			return nil, errors.New(string(body))
		}

		// Pharmacy names are HTML encoded for some reason, so decode them.
		for _, pharmacy := range pharmacies {
			pharmacy.Name = html.UnescapeString(pharmacy.Name)
		}

		return pharmacies, nil
	default:
		return nil, errors.New(string(body))
	}
}

func (c *Client) RequestQuote(ctx context.Context, params RequestQuoteParams) (*RequestQuoteResponse, error) {
	url := c.apiBaseURL + "/api/quote/QuoteRequest"
	const method = "POST"

	accessToken, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	payload, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to json serialize payload: %w", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create new request: %w", err)
	}

	req.Header.Add("Authorization", accessToken)
	req = req.WithContext(ctx)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute http request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	switch res.StatusCode {
	case http.StatusOK:
		var out *RequestQuoteResponse
		if err = json.Unmarshal(body, &out); err != nil {
			// If the json unmarshalling has failed, it could mean that the api has returned
			// an error message.
			var laproErr Error
			if err := json.Unmarshal(body, &laproErr); err != nil {
				return nil, fmt.Errorf("failed to parse response body as json: %s", string(body))
			}

			if laproErr.Success == "0" {
				return out, nil
			}

			// Bad status
			return nil, errors.New(string(body))
		}

		return out, nil

	default:
		return nil, errors.New(string(body))
	}
}

func (c *Client) GetCarrier(ctx context.Context, id int32) (*Carrier, error) {
	url := c.apiBaseURL + "/api/carrier/" + strconv.Itoa(int(id))
	const method = "GET"

	accessToken, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new request: %w", err)
	}

	req.Header.Add("Authorization", accessToken)
	req = req.WithContext(ctx)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute http request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	switch res.StatusCode {
	case http.StatusOK:
		var carrier Carrier
		if err = json.Unmarshal(body, &carrier); err != nil {

			// If the json unmarshalling has failed, it could mean that the api has returned
			// an error message.
			var laproErr Error
			if err := json.Unmarshal(body, &laproErr); err != nil {
				return nil, fmt.Errorf("failed to parse response body as json: %s", string(body))
			}

			if laproErr.Success == "0" {
				return nil, ErrNoResults
			}

			return nil, laproErr
		}
		return &carrier, nil
	default:
		return nil, fmt.Errorf("bad status code: %s", string(body))
	}
}

// GetAccessToken returns the current valid access token. If no access token exists, or it is expired, a new access token is retrieved.
func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()

	// When no access token is present (on first request), request an
	// access token and refresh token from the api
	if c.accessToken == "" {
		res, err := c.requestAccessToken(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to request access token: %w", err)
		}

		c.accessToken = res.AccessToken
		c.refreshToken = res.RefreshToken
		c.accessTokenExpiry = now.Add(time.Duration(res.ExpiresIn) * time.Second)

		return c.accessToken, nil
	}

	// When an access token is present, but it has expired, use the refresh token to
	// generate a new access token
	const safetyMargin = time.Duration(3) * time.Minute
	accessTokenExpiryWithSafetyMargin := c.accessTokenExpiry.Add(safetyMargin * -1)
	if now.After(accessTokenExpiryWithSafetyMargin) {
		res, err := c.requestAccessTokenWithRefresh(ctx, c.refreshToken)
		if err != nil {
			return "", fmt.Errorf("failed to request access token with refresh: %w", err)
		}

		c.accessToken = res.AccessToken
		c.refreshToken = res.RefreshToken
		c.accessTokenExpiry = now.Add(time.Duration(res.ExpiresIn) * time.Second)

		return c.accessToken, nil
	}

	// When the access token is present, and it is not expired, return it
	return c.accessToken, nil
}

// RequestAccessToken retrieves an access token & refresh token from LAPro using the supplied credentials.
func (c *Client) requestAccessToken(ctx context.Context) (*requestAccessTokenResponse, error) {
	url := c.authBaseURL + "/access_token"
	const method = "POST"

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	_ = writer.WriteField("username", c.username)
	_ = writer.WriteField("password", c.password)
	_ = writer.WriteField("grant_type", "password")
	_ = writer.WriteField("client_id", c.clientID)
	_ = writer.WriteField("client_secret", c.clientSecret)
	err := writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to write multipart payload: %w", err)
	}

	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create new request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = req.WithContext(ctx)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute http request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var out requestAccessTokenResponse
	err = json.Unmarshal(body, &out)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal json response body: %w", err)
	}

	return &out, nil
}

// RequestAccessTokenWithRefresh retrieves a new access token from LAPro using an existing refresh token.
func (c *Client) requestAccessTokenWithRefresh(ctx context.Context, refreshToken string) (*requestAccessTokenWithRefreshResponse, error) {
	url := c.authBaseURL + "/refresh_token"
	const method = "POST"

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	_ = writer.WriteField("refresh_token", refreshToken)
	_ = writer.WriteField("grant_type", "refresh_token")
	err := writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to write multipart payload: %w", err)
	}

	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create new request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = req.WithContext(ctx)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute http request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var out requestAccessTokenWithRefreshResponse
	err = json.Unmarshal(body, &out)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal json response body: %w", err)
	}

	return &out, nil
}
