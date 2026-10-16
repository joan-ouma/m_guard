package notification

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var arnPattern = regexp.MustCompile(`arn:aws:sns:[^\s"'>]+`)

type createPlatformEndpointResponse struct {
	XMLName xml.Name `xml:"CreatePlatformEndpointResponse"`
	Result  struct {
		EndpointArn string `xml:"EndpointArn"`
	} `xml:"CreatePlatformEndpointResult"`
}

type publishResponse struct {
	XMLName xml.Name `xml:"PublishResponse"`
	Result  struct {
		MessageID string `xml:"MessageId"`
	} `xml:"PublishResult"`
}

type snsErrorResponse struct {
	XMLName xml.Name `xml:"ErrorResponse"`
	Error   struct {
		Type    string `xml:"Type"`
		Code    string `xml:"Code"`
		Message string `xml:"Message"`
	} `xml:"Error"`
	RequestID string `xml:"RequestId"`
}

type AWSSNSClient struct {
	region          string
	accessKeyID     string
	secretAccessKey string
	endpointURL     string
	httpClient      *http.Client
}

func NewAWSSNSClient(region, accessKeyID, secretAccessKey string) *AWSSNSClient {
	endpoint := fmt.Sprintf("https://sns.%s.amazonaws.com", region)
	return &AWSSNSClient{
		region:          region,
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		endpointURL:     endpoint,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *AWSSNSClient) CreatePlatformEndpoint(ctx context.Context, platformAppARN, token, customUserData string) (string, error) {
	params := url.Values{}
	params.Set("Action", "CreatePlatformEndpoint")
	params.Set("Version", "2010-03-31")
	params.Set("PlatformApplicationArn", platformAppARN)
	params.Set("Token", token)
	if customUserData != "" {
		params.Set("CustomUserData", customUserData)
	}

	body, err := c.doRequest(ctx, params)
	if err != nil {
		existingARN := parseEndpointArnFromError(body)
		if existingARN != "" {
			_ = c.SetEndpointAttributes(ctx, existingARN, map[string]string{
				"Enabled": "true",
				"Token":   token,
			})
			return existingARN, nil
		}
		return "", fmt.Errorf("aws sns CreatePlatformEndpoint failed: %w", err)
	}

	var resp createPlatformEndpointResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to parse CreatePlatformEndpoint XML: %w", err)
	}

	return resp.Result.EndpointArn, nil
}

func (c *AWSSNSClient) SetEndpointAttributes(ctx context.Context, endpointARN string, attributes map[string]string) error {
	params := url.Values{}
	params.Set("Action", "SetEndpointAttributes")
	params.Set("Version", "2010-03-31")
	params.Set("EndpointArn", endpointARN)

	i := 1
	for k, v := range attributes {
		params.Set(fmt.Sprintf("Attributes.entry.%d.key", i), k)
		params.Set(fmt.Sprintf("Attributes.entry.%d.value", i), v)
		i++
	}

	_, err := c.doRequest(ctx, params)
	return err
}

func parseEndpointArnFromError(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var errResp snsErrorResponse
	if err := xml.Unmarshal(body, &errResp); err != nil {
		return ""
	}
	if !strings.Contains(errResp.Error.Message, "already exists with the same token") {
		return ""
	}
	return arnPattern.FindString(errResp.Error.Message)
}

func (c *AWSSNSClient) DeleteEndpoint(ctx context.Context, endpointARN string) error {
	params := url.Values{}
	params.Set("Action", "DeleteEndpoint")
	params.Set("Version", "2010-03-31")
	params.Set("EndpointArn", endpointARN)

	_, err := c.doRequest(ctx, params)
	if err != nil {
		return fmt.Errorf("aws sns DeleteEndpoint failed: %w", err)
	}
	return nil
}

func (c *AWSSNSClient) PublishToEndpoint(ctx context.Context, endpointARN string, subject, rawMessage, gcmPayload string) (string, error) {
	params := url.Values{}
	params.Set("Action", "Publish")
	params.Set("Version", "2010-03-31")
	params.Set("TargetArn", endpointARN)

	if subject != "" {
		params.Set("Subject", subject)
	}

	if gcmPayload != "" {
		// When targeting GCM/FCM, SNS accepts JSON message structure
		snsJSON := fmt.Sprintf(`{"default":%q,"GCM":%q}`, rawMessage, gcmPayload)
		params.Set("Message", snsJSON)
		params.Set("MessageStructure", "json")
	} else {
		params.Set("Message", rawMessage)
	}

	body, err := c.doRequest(ctx, params)
	if err != nil {
		return "", fmt.Errorf("aws sns Publish to endpoint failed: %w", err)
	}

	var resp publishResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to parse Publish XML: %w", err)
	}
	return resp.Result.MessageID, nil
}

func (c *AWSSNSClient) PublishSMS(ctx context.Context, phoneNumber, message string) (string, error) {
	params := url.Values{}
	params.Set("Action", "Publish")
	params.Set("Version", "2010-03-31")
	params.Set("PhoneNumber", phoneNumber)
	params.Set("Message", message)

	body, err := c.doRequest(ctx, params)
	if err != nil {
		return "", fmt.Errorf("aws sns Publish SMS failed: %w", err)
	}

	var resp publishResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to parse Publish SMS XML: %w", err)
	}
	return resp.Result.MessageID, nil
}

func (c *AWSSNSClient) PublishToTopic(ctx context.Context, topicARN string, subject, rawMessage, gcmPayload string) (string, error) {
	params := url.Values{}
	params.Set("Action", "Publish")
	params.Set("Version", "2010-03-31")
	params.Set("TopicArn", topicARN)

	if subject != "" {
		params.Set("Subject", subject)
	}

	if gcmPayload != "" {
		snsJSON := fmt.Sprintf(`{"default":%q,"GCM":%q}`, rawMessage, gcmPayload)
		params.Set("Message", snsJSON)
		params.Set("MessageStructure", "json")
	} else {
		params.Set("Message", rawMessage)
	}

	body, err := c.doRequest(ctx, params)
	if err != nil {
		return "", fmt.Errorf("aws sns Publish to topic failed: %w", err)
	}

	var resp publishResponse
	if err := xml.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to parse Publish XML: %w", err)
	}
	return resp.Result.MessageID, nil
}

// doRequest performs AWS SigV4 signed HTTP POST to AWS SNS Query API
func (c *AWSSNSClient) doRequest(ctx context.Context, params url.Values) ([]byte, error) {
	payload := params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpointURL+"/", strings.NewReader(payload))
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Host", "sns."+c.region+".amazonaws.com")
	req.Header.Set("X-Amz-Date", amzDate)

	// AWS SigV4 signing
	payloadHash := sha256Hex([]byte(payload))
	canonicalURI := "/"
	canonicalQuery := ""
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-amz-date:%s\n",
		req.Header.Get("Content-Type"),
		req.Header.Get("Host"),
		amzDate,
	)
	signedHeaders := "content-type;host;x-amz-date"

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		req.Method,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	)

	credentialScope := fmt.Sprintf("%s/%s/sns/aws4_request", dateStamp, c.region)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		amzDate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	)

	signingKey := getSignatureKey(c.secretAccessKey, dateStamp, c.region, "sns")
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		c.accessKeyID,
		credentialScope,
		signedHeaders,
		signature,
	)
	req.Header.Set("Authorization", authHeader)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var errResp snsErrorResponse
		if xml.Unmarshal(bodyBytes, &errResp) == nil && errResp.Error.Message != "" {
			return bodyBytes, fmt.Errorf("aws sns error [%s]: %s", errResp.Error.Code, errResp.Error.Message)
		}
		return bodyBytes, fmt.Errorf("http error %d: %s", res.StatusCode, string(bodyBytes))
	}

	return bodyBytes, nil
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key []byte, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func getSignatureKey(key, dateStamp, regionName, serviceName string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+key), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(regionName))
	kService := hmacSHA256(kRegion, []byte(serviceName))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}
