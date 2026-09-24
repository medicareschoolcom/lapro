package lapro

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

const (
	testStevenBrooksUserID = "35816"
)

type laProCredentials struct {
	username     string
	password     string
	clientID     string
	clientSecret string
}

func getLAProCredentials(t *testing.T) laProCredentials {
	c := laProCredentials{
		username:     os.Getenv("TEST_LAPRO_USERNAME"),
		password:     os.Getenv("TEST_LAPRO_PASSWORD"),
		clientID:     os.Getenv("TEST_LAPRO_CLIENT_ID"),
		clientSecret: os.Getenv("TEST_LAPRO_CLIENT_SECRET"),
	}

	if c.username == "" {
		t.Skip("set TEST_LAPRO_USERNAME to run this test ")
	}

	if c.password == "" {
		t.Skip("set TEST_LAPRO_PASSWORD to run this test ")
	}

	if c.clientID == "" {
		t.Skip("set TEST_LAPRO_CLIENT_ID to run this test ")
	}

	if c.clientSecret == "" {
		t.Skip("set TEST_LAPRO_CLIENT_SECRET to run this test ")
	}

	return c
}

func TestClient_requestAccessTokenWithRefresh(t *testing.T) {
	creds := getLAProCredentials(t)
	client := NewClient(DefaultAPIBaseURL, DefaultAuthBaseURL, creds.username, creds.password, creds.clientID, creds.clientSecret, &http.Client{})
	ctx := context.Background()

	resp, err := client.requestAccessToken(ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.requestAccessTokenWithRefresh(ctx, resp.RefreshToken)
	if err != nil {
		t.Fatalf("failed to request access token with refresh: %v", err)
	}
}

func TestClient_ListCountiesByZipCode(t *testing.T) {
	creds := getLAProCredentials(t)
	client := NewClient(DefaultAPIBaseURL, DefaultAuthBaseURL, creds.username, creds.password, creds.clientID, creds.clientSecret, &http.Client{})
	ctx := context.Background()

	tests := []struct {
		name        string
		zipCode     string
		expected    []*ListCountiesByZipCodeResponse
		expectedErr error
	}{
		{
			name:    "valid zip code with single county",
			zipCode: "90210",
			expected: []*ListCountiesByZipCodeResponse{
				{
					StateID:   "CA",
					StateName: "California",
					Counties: []County{
						{
							Name: "Los Angeles",
							Fips: "06037",
						},
					},
				},
			},
		},
		{
			name:    "valid zip code with multiple counties",
			zipCode: "57717",
			expected: []*ListCountiesByZipCodeResponse{
				{
					StateID:   "MT",
					StateName: "Montana",
					Counties: []County{
						{
							Name: "Carter",
							Fips: "30011",
						},
					},
				},
				{
					StateID:   "SD",
					StateName: "South Dakota",
					Counties: []County{
						{
							Name: "Butte",
							Fips: "46019",
						},
						{
							Name: "Harding",
							Fips: "46063",
						},
						{
							Name: "Lawrence",
							Fips: "46081",
						},
					},
				},
				{
					StateID:   "WY",
					StateName: "Wyoming",
					Counties: []County{
						{
							Name: "Crook",
							Fips: "56011",
						},
					},
				},
			},
		},
		{
			name:        "invalid zip code",
			zipCode:     "00000",
			expected:    nil,
			expectedErr: ErrNoResults,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := client.ListCountiesByZipCode(ctx, tt.zipCode)

			if tt.expectedErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.expectedErr)
				}
				if !errors.Is(err, tt.expectedErr) {
					t.Fatalf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if diff := cmp.Diff(tt.expected, response); diff != "" {
				t.Errorf("ListCountiesByZipCode() response mismatch (-expected +got):\n%s", diff)
			}
		})
	}
}

func TestClient_GetDrugNames(t *testing.T) {
	tt := []struct {
		name     string
		drugName string
		expected []*GetDrugNamesResponse
	}{
		{
			name:     "Drug names found",
			drugName: "Lip",
			expected: []*GetDrugNamesResponse{
				{DrugName: "Lipitor", DrugID: "13158"},
				{DrugName: "Lipofen", DrugID: "17669"},
			},
		},
		{
			name:     "Drug names search not case sensitive",
			drugName: "lip",
			expected: []*GetDrugNamesResponse{
				{DrugName: "Lipitor", DrugID: "13158"},
				{DrugName: "Lipofen", DrugID: "17669"},
			},
		},
		{
			name:     "Drug names not found",
			drugName: "Zzz",
			expected: []*GetDrugNamesResponse{},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			creds := getLAProCredentials(t)
			client := NewClient(DefaultAPIBaseURL, DefaultAuthBaseURL, creds.username, creds.password, creds.clientID, creds.clientSecret, &http.Client{})
			ctx := context.Background()

			resp, err := client.GetDrugNames(ctx, tc.drugName)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.expected, resp); diff != "" {
				t.Errorf("GetDrugNames() mismatch (-expected +got):\n%s", diff)
			}
		})
	}
}

func TestClient_ListDrugDosagesByDrugName(t *testing.T) {
	tt := []struct {
		name        string
		drugName    string
		expectedNum int
	}{
		{
			name:        "Returns Dosages",
			drugName:    "Lipitor",
			expectedNum: 4,
		},
		{
			name:        "Drug names are not case sensitive",
			drugName:    "lipitor",
			expectedNum: 4,
		},
		{
			name:        "Returns Humalog Dosages",
			drugName:    "Humalog",
			expectedNum: 17,
		},
		{
			name:        "Dosages not found",
			drugName:    "Zzz",
			expectedNum: 0,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			creds := getLAProCredentials(t)
			client := NewClient(DefaultAPIBaseURL, DefaultAuthBaseURL, creds.username, creds.password, creds.clientID, creds.clientSecret, &http.Client{})
			ctx := context.Background()

			drugs, err := client.ListDrugDosagesByDrugName(ctx, tc.drugName)
			if err != nil {
				t.Fatal(err)
			}

			if len(drugs) != tc.expectedNum {
				t.Errorf("number of drugs expected: %d, got: %d", tc.expectedNum, len(drugs))
			}
		})
	}
}

func TestClient_ListPharmaciesByZipCode(t *testing.T) {
	tt := []struct {
		name        string
		zipCode     string
		radius      string
		expectedNum int
	}{
		{
			name:        "returns pharmacies",
			zipCode:     "90210",
			radius:      "10",
			expectedNum: 50,
		},
		{
			name:        "bad zipCode",
			zipCode:     "00000",
			radius:      "10",
			expectedNum: 0,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			creds := getLAProCredentials(t)
			client := NewClient(DefaultAPIBaseURL, DefaultAuthBaseURL, creds.username, creds.password, creds.clientID, creds.clientSecret, &http.Client{})
			ctx := context.Background()

			pharmacies, err := client.ListPharmaciesByZipCode(ctx, tc.zipCode, tc.radius)
			if err != nil {
				t.Fatal(err)
			}

			if len(pharmacies) != tc.expectedNum {
				t.Errorf("number of pharmacies expected: %d, got: %d", tc.expectedNum, len(pharmacies))
			}
		})
	}
}

func TestClient_RequestQuote(t *testing.T) {
	creds := getLAProCredentials(t)
	client := NewClient(DefaultAPIBaseURL, DefaultAuthBaseURL, creds.username, creds.password, creds.clientID, creds.clientSecret, &http.Client{})
	ctx := context.Background()

	effectiveDate := getNextMonthFirstDay()

	tests := []struct {
		name        string
		params      RequestQuoteParams
		expectedNum int
		expectError bool
	}{
		{
			name: "empty drug list returns quotes",
			params: RequestQuoteParams{
				EffectiveDate: effectiveDate,
				Zip:           "90210",
				DOB:           "1960-01-01",
				County:        "LOS ANGELES",
				QuoteProducts: QuoteProducts{
					PartD:             true,
					MedicareAdvantage: false,
					MAPD:              false,
				},
				HealthStatus:      "Good",
				PharmacyID:        "53BJL",
				ExternalProviders: []ExternalProvider{},
				Drugs:             []QuoteDrug{},
			},
			expectedNum: 10,
			expectError: false,
		},
		{
			name: "drug list with atorvastatin returns quotes",
			params: RequestQuoteParams{
				EffectiveDate: effectiveDate,
				Zip:           "90210",
				DOB:           "1954-01-01",
				County:        "Los Angeles",
				Gender:        "M",
				QuoteProducts: QuoteProducts{
					PartD:             true,
					MedicareAdvantage: false,
					MAPD:              false,
				},
				HealthStatus:      "Good",
				PharmacyID:        "53BJL",
				ExternalProviders: []ExternalProvider{},
				Drugs: []QuoteDrug{
					{
						Name:      "atorvastatin calcium TAB 20MG",
						MetricQty: "30",
						Freq:      "30",
						NDCCode:   "60505257908",
					},
				},
			},
			expectedNum: 5,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.RequestQuote(ctx, tt.params)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got.MapDResults) < tt.expectedNum {
				t.Errorf("expected at least %d quotes, got %d", tt.expectedNum, len(got.MapDResults))
			}
		})
	}
}

func getNextMonthFirstDay() string {
	now := time.Now()
	nextMonth := now.AddDate(0, 1, 0)
	return time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, time.UTC).Format(time.DateOnly)
}

func TestClient_GetCarrier(t *testing.T) {
	tt := []struct {
		name      string
		carrierID int32
		expected  *Carrier
		expectErr bool
	}{
		{
			name:      "valid carrier ID",
			carrierID: 2678,
			expected: &Carrier{
				CarrierID: "2678",
				CompanyID: "bs_ca",
				ShortName: "BS_CA          ",
				Name:      "Blue Shield of California",
				Active:    "1",
				NAIC:      "61557",
			},
			expectErr: false,
		},
		{
			name:      "valid carrier ID 2",
			carrierID: 2712,
			expected: &Carrier{
				CarrierID: "2712",
				CompanyID: "aetnassrx",
				ShortName: "AETNASSRX      ",
				Name:      "Aetna-SilverScript",
				Active:    "1",
				NAIC:      "",
			},
			expectErr: false,
		},
		{
			name:      "invalid carrier ID",
			carrierID: 99999,
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			creds := getLAProCredentials(t)
			client := NewClient(DefaultAPIBaseURL, DefaultAuthBaseURL, creds.username, creds.password, creds.clientID, creds.clientSecret, &http.Client{})
			ctx := context.Background()

			carrier, err := client.GetCarrier(ctx, tc.carrierID)
			if tc.expectErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.expected, carrier); diff != "" {
				t.Errorf("GetCarrier() mismatch (-expected +got):\n%s", diff)
			}
		})
	}
}

func TestCent_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Cent
		wantErr  bool
	}{
		{
			name:     "valid currency string",
			input:    `"11.50"`,
			expected: 1150,
			wantErr:  false,
		},
		{
			name:     "currency with no decimals",
			input:    `"25"`,
			expected: 2500,
			wantErr:  false,
		},
		{
			name:     "zero value",
			input:    `"0.00"`,
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "empty string",
			input:    `""`,
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "null value",
			input:    `"null"`,
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "single cent",
			input:    `"0.01"`,
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "large amount",
			input:    `"123.45"`,
			expected: 12345,
			wantErr:  false,
		},
		{
			name:     "invalid format",
			input:    `"abc"`,
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "invalid currency",
			input:    `"12.34.56"`,
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "negative currency returns zero",
			input:    `"-0.20"`,
			expected: 0,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Cent
			err := c.UnmarshalJSON([]byte(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if c != tt.expected {
				t.Errorf("expected %d cents, got %d cents", tt.expected, c)
			}
		})
	}
}
