package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/eolymp/contracts/go/eolymp/atlas"
	"github.com/eolymp/contracts/go/eolymp/judge"
	"github.com/eolymp/contracts/go/eolymp/playground"
	"github.com/eolymp/go-packages/env"
	"github.com/eolymp/go-packages/httpx"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type Client struct {
	cli httpx.Client
}

func (c *Client) CreateObject(ctx context.Context, in *atlas.CreateObjectInput) (*atlas.CreateObjectOutput, error) {
	out := &atlas.CreateObjectOutput{}

	if err := c.Invoke(ctx, "eolymp.atlas.Atlas/CreateObject", in, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *Client) CreateProblem(ctx context.Context, in *judge.CreateProblemInput) (*judge.CreateProblemOutput, error) {
	out := &judge.CreateProblemOutput{}

	if err := c.Invoke(ctx, "eolymp.judge.Judge/CreateProblem", in, out); err != nil {
		return nil, err
	}

	return out, nil
}
func (c *Client) DescribeSubmission(ctx context.Context, in *judge.DescribeSubmissionInput) (*judge.DescribeSubmissionOutput, error) {
	out := &judge.DescribeSubmissionOutput{}

	if err := c.Invoke(ctx, "eolymp.judge.Judge/DescribeSubmission", in, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *Client) CreateSubmission(ctx context.Context, in *judge.CreateSubmissionInput) (*judge.CreateSubmissionOutput, error) {
	out := &judge.CreateSubmissionOutput{}

	if err := c.Invoke(ctx, "eolymp.judge.Judge/CreateSubmission", in, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *Client) DescribeRun(ctx context.Context, in *playground.DescribeRunInput) (*playground.DescribeRunOutput, error) {
	out := &playground.DescribeRunOutput{}

	if err := c.Invoke(ctx, "eolymp.playground.Playground/DescribeRun", in, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *Client) CreateRun(ctx context.Context, in *playground.CreateRunInput) (*playground.CreateRunOutput, error) {
	out := &playground.CreateRunOutput{}

	if err := c.Invoke(ctx, "eolymp.playground.Playground/CreateRun", in, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *Client) Invoke(ctx context.Context, method string, in, out proto.Message) (err error) {
	base := strings.TrimSuffix(env.StringDefault("EOLYMP_API_URL", "https://api.e-olymp.com"), "/")

	var input io.Reader
	if in != nil {
		data, err := protojson.Marshal(in)
		if err != nil {
			return err
		}

		input = bytes.NewReader(data)
	}

	req, err := http.NewRequest(http.MethodPost, base+"/twirp/"+method, input)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	req = req.WithContext(ctx)

	resp, err := c.cli.Do(req)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("non-200 response code (%v)", resp.StatusCode)
	}

	if out != nil {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if err := protojson.Unmarshal(data, out); err != nil {
			return err
		}
	}

	return nil
}
