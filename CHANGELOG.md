# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.0.0] - 2026-09-24

### Added
- `DefaultAPIBaseURL` and `DefaultAuthBaseURL` constants for the production LAPro endpoints

### Changed
- **Breaking:** Module path is now `github.com/medicareschoolcom/lapro/v2`
- **Breaking:** `NewClient()` now takes `apiBaseURL` and `authBaseURL` as its first two arguments instead of hardcoding the LAPro URLs

## [1.0.0] - 2025-09-22

### Added
- Initial release of LAPro Go Client library
- OAuth2 authentication with automatic token management and refresh
- Geographic data API methods:
  - `ListCountiesByZipCode()` - Retrieve counties by zip code
- Drug information API methods:
  - `GetDrugNames()` - Search drug names by partial match
  - `ListDrugDosagesByDrugName()` - Get dosages for a specific drug
- Pharmacy lookup API methods:
  - `ListPharmaciesByZipCode()` - Find pharmacies within radius of zip code
- Insurance quote API methods:
  - `RequestQuote()` - Request Medicare Part D and Medicare Advantage quotes
- Carrier information API methods:
  - `GetCarrier()` - Get insurance carrier details by ID
- Custom types for proper JSON unmarshaling:
  - `Boolean` type - Handles non-standard boolean representations ("yes"/"no", "1"/"0")
  - `Cent` type - Converts currency strings to integer cents for precise calculations
- Core data structures:
  - `Drug` - Drug information with NDC codes, strengths, and dosages
  - `Pharmacy` - Pharmacy details with location and contact information
  - `Quote` - Insurance quote with premiums and cost estimates
  - `County` - Geographic county information with FIPS codes
  - `Carrier` - Insurance carrier information
- Thread-safe client implementation with mutex-protected token management
- Comprehensive error handling with `ErrNoResults` for empty API responses
- Health check functionality via `Ping()` method
- Comprehensive integration test suite covering all API methods
- MIT License
- Documentation with usage examples and API reference

### Security
- Secure OAuth2 credential handling
- Automatic access token refresh with safety margins
- Thread-safe concurrent access protection