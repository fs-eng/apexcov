package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli"
)

// main is the entry point for the apexcov CLI application
func main() {
	app := cli.NewApp()
	app.Usage = "a Test Coverage Generator for Apex"
	app.Version = "1.0.0"
	// app.Author = "Jean-Philippe Monette"
	// app.Email = "contact@jpmonette.net"

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "instance,i",
			Value: "https://login.salesforce.com",
			Usage: "instance to use",
		},
		&cli.StringFlag{
			Name:  "username,u",
			Value: os.Getenv("APEXCOV_USERNAME"),
			Usage: "username of the Salesforge org",
		},
		&cli.StringFlag{
			Name:  "password,p",
			Value: os.Getenv("APEXCOV_PASSWORD"),
			Usage: "password of the Salesforge org",
		},
	}

	app.Action = apexcov
	app.Run(os.Args)
}

// apexcov handles the code coverage command
func apexcov(c *cli.Context) error {
	username := c.String("username")
	password := c.String("password")
	instance := c.String("instance")

	if os.Getenv("APEXCOV_INSTANCE") != "" {
		instance = os.Getenv("APEXCOV_INSTANCE")
	}

	if username == "" {
		return cli.NewExitError("You must provide a username", 1)
	} else if password == "" {
		return cli.NewExitError("You must provide a password", 1)
	} else if _, err := url.ParseRequestURI(instance); err != nil {
		return cli.NewExitError("You must provide a valid instance URL", 1)
	}

	instanceURL, sessionID, err := login(instance, username, password)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	data, err := getCoverage(instanceURL, sessionID)

	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	body := "TN:\n"

	dir, err := os.Getwd()

	for _, class := range data.Records {

		// Determine if the file is a class or trigger and create filePath accordingly.
		isClass := strings.HasPrefix(class.ID, "01p")
		filePath := ""

		if isClass {
			filePath = dir + "/classes/" + class.ApexClassOrTrigger.Name + ".cls"
		} else {
			filePath = dir + "/triggers/" + class.ApexClassOrTrigger.Name + ".trigger"
		}

		// Only send data to the body output if the file exists in the local sandbox.
		// This will prevent errors when running the test-reporter format-coverage command.
		if fileExists(filePath) {
			if isClass {
				body += "SF:" + filePath + "\n"
			} else {
				body += "SF:" + filePath + "\n"
			}

			for _, line := range class.Coverage.CoveredLines {
				body += "DA:" + strconv.Itoa(line) + ",1\n"
			}

			for _, line := range class.Coverage.UncoveredLines {
				body += "DA:" + strconv.Itoa(line) + ",0\n"
			}

			body += "end_of_record\n"
		}
	}

	persistCoverage(body)
	return nil
}

// getCoverage gets the Apex code coverage from the Salesforce instance
func getCoverage(instanceURL, session string) (coverage CoverageResponse, err error) {
	client := &http.Client{}

	endpoint := instanceURL + "/services/data/v39.0/tooling/query?q="
	query := "SELECT ApexClassOrTriggerId, ApexClassorTrigger.Name, Coverage FROM ApexCodeCoverageAggregate WHERE NOT ApexClassorTrigger.Name = null"

	req, err := http.NewRequest("GET", endpoint+url.QueryEscape(query), nil)
	req.Header.Add("Authorization", "Bearer "+session)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "apexcov")
	response, err := client.Do(req)

	if err != nil {
		return coverage, err
	}

	defer response.Body.Close()

	responseData, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return coverage, err
	}

	err = json.Unmarshal(responseData, &coverage)

	if err != nil {
		return coverage, err
	}
	return
}

// persistCoverage stores the coverage in the lcov.info file
func persistCoverage(body string) error {
	_, err := os.Stat("./coverage")
	if os.IsNotExist(err) {
		os.Mkdir("./coverage", 0777)
	}

	err = ioutil.WriteFile("./coverage/lcov.info", []byte(body), 0666)
	return err
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// CoverageResponse represents the format of the ApexCodeCoverageAggregate query response
type CoverageResponse struct {
	Records []struct {
		ID                 string `json:"ApexClassOrTriggerId"`
		ApexClassOrTrigger struct {
			Name string `json:"Name"`
		} `json:"ApexClassOrTrigger"`
		Coverage struct {
			CoveredLines   []int `json:"coveredLines"`
			UncoveredLines []int `json:"uncoveredLines"`
		} `json:"Coverage"`
	} `json:"records"`
}
