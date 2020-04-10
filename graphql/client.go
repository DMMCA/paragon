package graphql

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kcarretto/paragon/ent"
	"github.com/kcarretto/paragon/graphql/generated"
	"github.com/kcarretto/paragon/graphql/models"
	"github.com/kcarretto/paragon/pkg/auth"
)

// QueryResolver is an alias for github.com/kcarretto/paragon/graphql/generated.QueryResolver
type QueryResolver generated.QueryResolver

// MutationResolver is an alias for github.com/kcarretto/paragon/graphql/generated.MutationResolver
type MutationResolver generated.MutationResolver

// An Error returned by the GraphQL server
type Error struct {
	Message string   `json:"message"`
	Path    []string `json:"path"`
}

// Error implements the error interface by formatting an error message of the available error info.
func (err Error) Error() string {
	return fmt.Sprintf("%s (path: %s)", err.Message, strings.Join(err.Path, ", "))
}

// A Request stores execution properties of GraphQL queries and mutations.
type Request struct {
	Operation string      `json:"operationName"`
	Query     string      `json:"query"`
	Variables interface{} `json:"variables,omitempty"`
}

// A Client can be used to request GraphQL queries and mutations using HTTP.
type Client struct {
	URL  string
	HTTP *http.Client

	Service    string
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// Do executes a GraphQL request and unmarshals the JSON result into the destination struct.
func (client *Client) Do(ctx context.Context, request Request, dst interface{}) error {
	// Encode request payload
	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to encode json: %w", err)
	}

	// Build http request
	httpReq, err := http.NewRequest(http.MethodPost, client.URL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Sign http request
	epoch := fmt.Sprintf("%d", time.Now().Unix())
	sig, err := client.sign([]byte(epoch))
	if err != nil {
		panic(fmt.Errorf("failed to sign request: %w", err))
	}
	httpReq.Header.Set(auth.HeaderService, client.Service)
	httpReq.Header.Set(auth.HeaderIdentity, base64.StdEncoding.EncodeToString(client.PublicKey))
	httpReq.Header.Set(auth.HeaderEpoch, epoch)
	httpReq.Header.Set(auth.HeaderSignature, base64.StdEncoding.EncodeToString(sig))

	// Set default http client if necessary
	if client.HTTP == nil {
		client.HTTP = http.DefaultClient
	}

	// Issue the request
	httpResp, err := client.HTTP.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer httpResp.Body.Close()

	// Check response status
	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status error: %s", httpResp.Status)
	}

	tstData, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		panic(err)
	}

	// Decode the response
	// data := json.NewDecoder(httpResp.Body)
	data := json.NewDecoder(bytes.NewBuffer(tstData))
	if err := data.Decode(dst); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// ClaimTasks for a target that has the provided attributes, returning an array of tasks to execute.
// If no tasks are available, an empty task array is returned. If no target can be found, an error
// will be returned.
func (client *Client) ClaimTasks(ctx context.Context, vars models.ClaimTasksRequest) ([]*ent.Task, error) {
	// Build request
	req := Request{
		Operation: "ClaimTasks",
		Query: `
		mutation ClaimTasks($params: ClaimTasksRequest!) {
			claimTasks(input: $params) {
			  id
			  content
			}
		}`,
		Variables: map[string]interface{}{
			"params": vars,
		},
	}

	// Prepare response
	var resp struct {
		Data struct {
			Tasks []*ent.Task `json:"claimTasks"`
		} `json:"data"`
		Errors []Error `json:"errors"`
	}

	// Execute mutation
	if err := client.Do(ctx, req, &resp); err != nil {
		return nil, err
	}

	fmt.Printf("Response from teamserver: %+v\n", resp)

	// Check for errors
	if resp.Errors != nil {
		return nil, fmt.Errorf("mutation failed: [%+v]", resp.Errors)
	}

	// Return claimed tasks
	return resp.Data.Tasks, nil
}

// ClaimTask with the provided ID.
func (client *Client) ClaimTask(ctx context.Context, id int) (*ent.Task, error) {
	// Build request
	req := Request{
		Operation: "ClaimTask",
		Query: `
		mutation ClaimTask($id: ID!) {
			claimTask(id: $id) {
			  id
			  content
			}
		}`,
		Variables: map[string]interface{}{
			"id": id,
		},
	}

	// Prepare response
	var resp struct {
		Data struct {
			Task *ent.Task `json:"claimTask"`
		} `json:"data"`
		Errors []Error `json:"errors"`
	}

	// Execute mutation
	if err := client.Do(ctx, req, &resp); err != nil {
		return nil, err
	}

	fmt.Printf("Response from teamserver: %+v\n", resp)

	// Check for errors
	if resp.Errors != nil {
		return nil, fmt.Errorf("mutation failed: [%+v]", resp.Errors)
	}

	// Return claimed task
	return resp.Data.Task, nil
}

// SubmitTaskResult updates a task with execution output.
func (client *Client) SubmitTaskResult(ctx context.Context, vars models.SubmitTaskResultRequest) error {
	// Build request
	req := Request{
		Operation: "SubmitTaskResult",
		Query: `
		mutation SubmitTaskResult($params: SubmitTaskResultRequest!) {
			submitTaskResult(input: $params) {
			  id
			}
		}`,
		Variables: map[string]interface{}{
			"params": vars,
		},
	}

	// Prepare response
	var resp struct {
		Errors []Error `json:"errors"`
	}

	// Execute mutation
	if err := client.Do(ctx, req, &resp); err != nil {
		return err
	}

	// Check for errors
	if resp.Errors != nil {
		return fmt.Errorf("mutation failed: [%+v]", resp.Errors)
	}

	return nil
}

// CreateTarget creates a target, but may error if it already exists.
func (client *Client) CreateTarget(ctx context.Context, vars models.CreateTargetRequest) (*ent.Target, error) {
	// Build request
	req := Request{
		Operation: "CreateTarget",
		Query: `
		mutation CreateTarget($params: CreateTargetRequest!) {
			target: createTarget(input: $params) {
				id
				name
				primaryIP
			}
		}`,
		Variables: map[string]interface{}{
			"params": vars,
		},
	}

	// Prepare response
	var resp struct {
		Data struct {
			Target *ent.Target `json:"target"`
		} `json:"data"`
		Errors []Error `json:"errors"`
	}

	// Execute mutation
	if err := client.Do(ctx, req, &resp); err != nil {
		return nil, err
	}

	// Check for errors
	if resp.Errors != nil {
		return nil, fmt.Errorf("mutation failed: [%+v]", resp.Errors)
	}
	if resp.Data.Target == nil {
		return nil, fmt.Errorf("no target data returned from mutation")
	}

	return resp.Data.Target, nil
}

// ListTags provides a map of name to tag for all existing tags.
func (client *Client) ListTags(ctx context.Context) (map[string]*ent.Tag, error) {
	// Build request
	req := Request{
		Operation: "ListTags",
		Query: `
		query ListTags {
			tags {
			  id
			  name
			}
		}`,
		Variables: map[string]interface{}{},
	}

	// Prepare response
	var resp struct {
		Data struct {
			Tags []*ent.Tag `json:"tags"`
		} `json:"data"`
	}

	// Execute mutation
	if err := client.Do(ctx, req, &resp); err != nil {
		return nil, err
	}

	// Collect non-nil tags into map
	tags := make(map[string]*ent.Tag, len(resp.Data.Tags))
	for _, tag := range resp.Data.Tags {
		if tag == nil {
			continue
		}

		tags[tag.Name] = tag
	}

	return tags, nil
}

// CreateTag will create a tag, but may error if the tag already exists.
func (client *Client) CreateTag(ctx context.Context, vars models.CreateTagRequest) (*ent.Tag, error) {
	// Build request
	req := Request{
		Operation: "CreateTag",
		Query: `
		mutation CreateTag($params: CreateTagRequest!) {
			tag: createTag(input: $params) {
			  id
			  name
			}
		}`,
		Variables: map[string]interface{}{
			"params": vars,
		},
	}

	// Prepare response
	var resp struct {
		Data struct {
			Tag *ent.Tag `json:"tag"`
		} `json:"data"`
		Errors []Error `json:"errors"`
	}

	// Execute mutation
	if err := client.Do(ctx, req, &resp); err != nil {
		return nil, err
	}

	// Check for errors
	if resp.Errors != nil {
		return nil, fmt.Errorf("mutation failed: [%+v]", resp.Errors)
	}
	if resp.Data.Tag == nil {
		return nil, fmt.Errorf("no tag data returned from mutation")
	}

	return resp.Data.Tag, nil
}

// CreateTags ensures that the provided set of tags exists and creates any that do not yet exist. If
// duplicate tag names are provided, the tag will only be created once.
func (client *Client) CreateTags(ctx context.Context, names ...string) (map[string]*ent.Tag, error) {
	// Prevent query if no names are specified
	if len(names) < 1 {
		return make(map[string]*ent.Tag, 0), nil
	}

	// Get a map of existing tags
	tagMap, err := client.ListTags(ctx)
	if err != nil {
		return nil, err
	}

	// Create tags that don't exist yet
	for _, name := range names {
		// Skip if the tag already exists
		if _, exists := tagMap[name]; exists {
			continue
		}

		// Otherwise, create it
		tag, err := client.CreateTag(ctx, models.CreateTagRequest{
			Name: name,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create tag %q: %w", name, err)
		}

		// Prevent it from being created again
		tagMap[name] = tag
	}

	return tagMap, nil
}

func (client *Client) sign(msg []byte) ([]byte, error) {
	// If nil, try loading from the environment
	if client.PublicKey == nil || client.PrivateKey == nil {
		client.PublicKey, client.PrivateKey = client.keyFromEnv()
	}

	// Still nil, generate a keypair
	if client.PublicKey == nil || client.PrivateKey == nil {
		pubKey, privKey, err := ed25519.GenerateKey(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to generate keypair: %w", err)
		}
		client.PublicKey = pubKey
		client.PrivateKey = privKey
	}

	// Sign the message using the client's private key
	return client.PrivateKey.Sign(nil, msg, crypto.Hash(0))
}

func (client *Client) keyFromEnv() (ed25519.PublicKey, ed25519.PrivateKey) {
	if key := os.Getenv("PG_SVC_KEY"); key != "" {
		parts := strings.Split(key, ":")
		if len(parts) != 2 {
			panic(fmt.Errorf("invalid format for PG_SVC_KEY, expected b64PubKey:b64PrivKey"))
		}
		pubKey, err := base64.StdEncoding.DecodeString(parts[0])
		if err != nil {
			panic(fmt.Errorf("invalid base64 provided for PubKey in PG_SVC_KEY: %w", err))
		}
		privKey, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			panic(fmt.Errorf("invalid base64 provided for PrivKey in PG_SVC_KEY: %w", err))
		}
		return pubKey, privKey
	}
	return nil, nil
}
