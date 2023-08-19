package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
)

type NativeName struct {
	Common   string `json:"common"`
	Official string `json:"official"`
}

type Country struct {
	Name       NativeName        `json:"name"`
	Capital    []string          `json:"capital"`
	Population int               `json:"population"`
	Region     string            `json:"region"`
	Language   map[string]string `json:"languages"`
	Area       float64           `json:"area"`
}

func apiKeyMiddleware(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve the API key from the request headers or query parameters
		clientAPIKey := c.GetHeader("X-API-Key")
		if clientAPIKey == "" {
			clientAPIKey = c.Query("apiKey")
		}

		// Validate the API key
		if clientAPIKey != apiKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Continue to the next middleware or route
		c.Next()
	}
}

func fetchFilteredCountries(populationFrom, populationTo, areaFrom, areaTo int, language, sortBy string, ascending bool, pageStr, pageSizeStr string) ([]Country, error) {
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		return nil, fmt.Errorf("invalid page number")
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid page size")
	}
	apiURL := "https://restcountries.com/v3.1/all"
	response, err := http.Get(apiURL)
	if err != nil {
		fmt.Println("Error making the request:", err)
		return nil, err
	}
	defer response.Body.Close()

	var countries []Country
	err = json.NewDecoder(response.Body).Decode(&countries)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return nil, err
	}
	var areato float64 = float64(areaTo)
	var areafrom float64 = float64(areaFrom)
	// Apply filters
	filteredCountries := make([]Country, 0)
	for _, country := range countries {
		// Apply population filter
		if populationFrom >= 0 && country.Population < populationFrom {
			continue
		}
		if populationTo > 0 && country.Population > populationTo {
			continue
		}
		// Apply language filter
		if areafrom > 0 && country.Area < areafrom {
			continue
		}
		if areato > 0 && country.Area > areato {
			continue
		}

		// Apply language filter

		if language != "" {
			// Check if the provided language matches any of the country's languages
			languageMatch := false
			for _, lang := range country.Language {
				if lang == language {
					languageMatch = true
					break
				}
			}
			if !languageMatch {
				continue
			}
		}

		filteredCountries = append(filteredCountries, country)
	}

	// Apply sorting
	sort.Slice(filteredCountries, func(i, j int) bool {
		if ascending {
			return filteredCountries[i].Name.Common < filteredCountries[j].Name.Common
		}
		return filteredCountries[i].Name.Common > filteredCountries[j].Name.Common
	})

	// Apply pagination
	startIndex := (page - 1) * pageSize
	endIndex := startIndex + pageSize
	if endIndex > len(filteredCountries) {
		endIndex = len(filteredCountries)
	}
	return filteredCountries[startIndex:endIndex], nil
}

func fetchCountryData(name string) (Country, error) {
	apiURL := "https://restcountries.com/v3.1/name/" + name
	response, err := http.Get(apiURL)
	if err != nil {
		fmt.Println("Error making the request:", err)
		return Country{}, err
	}
	defer response.Body.Close()
	var countries []Country
	err = json.NewDecoder(response.Body).Decode(&countries)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return Country{}, err
	}
	if len(countries) > 0 {
		return countries[0], nil
	}

	return Country{}, fmt.Errorf("country not found")

}

func main() {

	// Generate a unique API key
	apiKey := uuid.New().String()

	fmt.Println("Generated API Key:", apiKey)

	router := gin.Default()

	router.GET("/countries/:name", apiKeyMiddleware(apiKey), func(c *gin.Context) {
		countryName := c.Param("name")

		country, err := fetchCountryData(countryName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, country)
	})
	router.GET("/countries", apiKeyMiddleware(apiKey), func(c *gin.Context) {
		populationFrom, _ := strconv.Atoi(c.DefaultQuery("populationFrom", "0"))
		populationTo, _ := strconv.Atoi(c.DefaultQuery("populationTo", "10000000000000"))
		areaFrom, _ := strconv.Atoi(c.DefaultQuery("areaFrom", "0"))
		areaTo, _ := strconv.Atoi(c.DefaultQuery("areaTo", "100000000000000000000000000000000"))

		language := c.Query("language")
		sortBy := c.DefaultQuery("sortBy", "name") // Default sorting by name
		ascending := c.DefaultQuery("ascending", "true") == "true"
		pageStr := c.DefaultQuery("page", "1")
		pageSizeStr := c.DefaultQuery("pageSize", "10")

		// Convert query parameters to appropriate types if needed

		countries, err := fetchFilteredCountries(populationFrom, populationTo, areaFrom, areaTo, language, sortBy, ascending, pageStr, pageSizeStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, countries)
	})

	router.Run(":8080")
}
