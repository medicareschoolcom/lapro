# LAPro Go Client

A Go client library for the Lead Advantage Pro (LAPro) API, providing access to Medicare insurance quotes, drug information, pharmacy locations, and geographical data.

## Features

- **Authentication**: Automatic OAuth2 token management with refresh
- **Drug Information**: Search for drug names and dosages
- **Pharmacy Lookup**: Find pharmacies by zip code and radius
- **Geographic Data**: Retrieve counties by zip code
- **Insurance Quotes**: Request Medicare Part D and Medicare Advantage quotes
- **Carrier Information**: Get insurance carrier details
- **Custom Types**: Built-in support for currency (Cent) and boolean unmarshaling

## Installation

```bash
go get github.com/medicareschoolcom/lapro/v2
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "net/http"

    "github.com/medicareschoolcom/lapro/v2"
)

func main() {
    client := lapro.NewClient(
        lapro.DefaultAPIBaseURL,
        lapro.DefaultAuthBaseURL,
        "your-username",
        "your-password",
        "your-client-id",
        "your-client-secret",
        &http.Client{},
    )

    ctx := context.Background()

    // Check client health
    if err := client.Ping(ctx); err != nil {
        fmt.Printf("Client health check failed: %v\n", err)
        return
    }

    // Get counties for a zip code
    counties, err := client.ListCountiesByZipCode(ctx, "90210")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Found %d state(s) for zip code 90210\n", len(counties))
}
```

## API Methods

### Authentication
- `Ping(ctx)` - Health check that verifies authentication

### Geographic Data
- `ListCountiesByZipCode(ctx, zipCode)` - Get counties by zip code

### Drug Information
- `GetDrugNames(ctx, searchTerm)` - Search drug names by partial match
- `ListDrugDosagesByDrugName(ctx, drugName)` - Get dosages for a drug

### Pharmacy Lookup
- `ListPharmaciesByZipCode(ctx, zipCode, radius)` - Find pharmacies within radius

### Insurance Quotes
- `RequestQuote(ctx, params)` - Request Medicare insurance quotes

### Carrier Information
- `GetCarrier(ctx, carrierID)` - Get insurance carrier details

## Data Types

### Custom Types

The library includes custom types for proper JSON unmarshaling:

- **`Boolean`** - Handles non-standard boolean representations ("yes"/"no", "1"/"0")
- **`Cent`** - Converts currency strings to integer cents for precise calculations

### Core Structures

- **`Drug`** - Drug information including NDC codes, strengths, and dosages
- **`Pharmacy`** - Pharmacy details with location and contact information
- **`Quote`** - Insurance quote with premiums and cost estimates
- **`County`** - Geographic county information with FIPS codes

## Error Handling

The client returns `lapro.ErrNoResults` when API calls succeed but return no data. Other errors include detailed API response information.

```go
counties, err := client.ListCountiesByZipCode(ctx, "00000")
if err != nil {
    if errors.Is(err, lapro.ErrNoResults) {
        fmt.Println("No counties found for this zip code")
    } else {
        fmt.Printf("API error: %v\n", err)
    }
}
```

## Testing

The library includes comprehensive integration tests. Set the following environment variables to run tests:

```bash
export TEST_LAPRO_USERNAME="your-test-username"
export TEST_LAPRO_PASSWORD="your-test-password"
export TEST_LAPRO_CLIENT_ID="your-test-client-id"
export TEST_LAPRO_CLIENT_SECRET="your-test-client-secret"

go test -v
```

## Thread Safety

The client is thread-safe. Access tokens are automatically managed with proper mutex protection for concurrent usage.

## License

See LICENSE file for details.
