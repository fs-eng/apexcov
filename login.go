package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// login executes the login to the SOAP API and returns the Instance URL and Session ID
func login(instance, username, password string) (instanceURL, sessionID string, err error) {
	client := &http.Client{}

	soap := `
		<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/"
  				xmlns:urn="urn:partner.soap.sforce.com">
  			<soapenv:Body>
    			<urn:login>
      				<urn:username>%s</urn:username>
      				<urn:password>%s</urn:password>
    			</urn:login>
  			</soapenv:Body>
		</soapenv:Envelope>
		`

	rbody := fmt.Sprintf(soap, username, password)

	req, err := http.NewRequest("POST", instance+"/services/Soap/u/39.0", strings.NewReader(rbody))
	req.Header.Add("Content-Type", `text/xml`)
	req.Header.Add("SOAPAction", `login`)
	response, err := client.Do(req)

	if err != nil {
		return
	}

	defer response.Body.Close()

	if response.StatusCode == 401 {
		err = errors.New("Unauthorized")
		return
	}

	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return
	}

	err = processError(body)

	if err != nil {
		return
	}

	var loginResponse SoapLoginResponse

	if err = xml.Unmarshal(body, &loginResponse); err != nil {
		return
	}

	u, err := url.Parse(loginResponse.InstanceURL)
	sessionID = loginResponse.SessionID
	instanceURL = "https://" + u.Host

	return
}

// processError process the error returned by the SOAP API
func processError(body []byte) (err error) {
	var soapError SoapErrorResponse
	xml.Unmarshal(body, &soapError)
	if soapError.FaultCode != "" {
		return errors.New(soapError.FaultString)
	}
	return
}

// SoapLoginResponse represents the response of the "login" SOAPAction
type SoapLoginResponse struct {
	SessionID   string `xml:"Body>loginResponse>result>sessionId"`
	ID          string `xml:"Body>loginResponse>result>userId"`
	InstanceURL string `xml:"Body>loginResponse>result>serverUrl"`
}

// SoapErrorResponse represents the error response of the SOAP API
type SoapErrorResponse struct {
	FaultCode   string `xml:"Body>Fault>faultcode"`
	FaultString string `xml:"Body>Fault>faultstring"`
}
