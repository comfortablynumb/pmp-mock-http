package management

import (
	"time"

	"github.com/comfortablynumb/pmp-mock-http/internal/models"
)

// LoadDefaultTemplates loads pre-built templates for common APIs
func LoadDefaultTemplates(manager *Manager) error {
	templates := []MockTemplate{
		// Stripe API Templates
		{
			ID:          "stripe-create-payment",
			Name:        "Stripe - Create Payment Intent",
			Description: "Mock Stripe payment intent creation",
			Category:    TemplateStripe,
			Tags:        []string{"stripe", "payment", "post"},
			Mock: models.Mock{
				Name:     "Stripe Create Payment Intent",
				Priority: 10,
				Request: models.Request{
					URI:    "/v1/payment_intents",
					Method: "POST",
				},
				Response: models.Response{
					StatusCode: 200,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
					Body: `{
						"id": "pi_{{randomString 16}}",
						"object": "payment_intent",
						"amount": 2000,
						"currency": "usd",
						"status": "requires_payment_method",
						"created": {{timestamp}}
					}`,
					Template: true,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:          "stripe-retrieve-customer",
			Name:        "Stripe - Retrieve Customer",
			Description: "Mock Stripe customer retrieval",
			Category:    TemplateStripe,
			Tags:        []string{"stripe", "customer", "get"},
			Mock: models.Mock{
				Name:     "Stripe Retrieve Customer",
				Priority: 10,
				Request: models.Request{
					URI:    "/v1/customers/.*",
					Method: "GET",
					IsRegex: models.RegexConfig{
						URI: true,
					},
				},
				Response: models.Response{
					StatusCode: 200,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
					Body: `{
						"id": "cus_{{randomString 14}}",
						"object": "customer",
						"email": "{{email}}",
						"name": "{{fullName}}",
						"created": {{timestamp}}
					}`,
					Template: true,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},

		// GitHub API Templates
		{
			ID:          "github-list-repos",
			Name:        "GitHub - List Repositories",
			Description: "Mock GitHub repository listing",
			Category:    TemplateGitHub,
			Tags:        []string{"github", "repositories", "get"},
			Mock: models.Mock{
				Name:     "GitHub List Repositories",
				Priority: 10,
				Request: models.Request{
					URI:    "/user/repos",
					Method: "GET",
				},
				Response: models.Response{
					StatusCode: 200,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
					Body: `[
						{
							"id": {{randomInt 100000}},
							"name": "example-repo",
							"full_name": "user/example-repo",
							"private": false,
							"html_url": "https://github.com/user/example-repo",
							"description": "An example repository",
							"created_at": "{{timestamp}}",
							"updated_at": "{{timestamp}}"
						}
					]`,
					Template: true,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:          "github-create-issue",
			Name:        "GitHub - Create Issue",
			Description: "Mock GitHub issue creation",
			Category:    TemplateGitHub,
			Tags:        []string{"github", "issues", "post"},
			Mock: models.Mock{
				Name:     "GitHub Create Issue",
				Priority: 10,
				Request: models.Request{
					URI:    "/repos/.*/issues",
					Method: "POST",
					IsRegex: models.RegexConfig{
						URI: true,
					},
				},
				Response: models.Response{
					StatusCode: 201,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
					Body: `{
						"id": {{randomInt 100000}},
						"number": {{randomInt 1000}},
						"title": "{{.Request.JSONPath "$.title"}}",
						"state": "open",
						"created_at": "{{timestamp}}",
						"updated_at": "{{timestamp}}"
					}`,
					Template: true,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},

		// AWS S3 Templates
		{
			ID:          "aws-s3-put-object",
			Name:        "AWS S3 - Put Object",
			Description: "Mock AWS S3 object upload",
			Category:    TemplateAWS,
			Tags:        []string{"aws", "s3", "put"},
			Mock: models.Mock{
				Name:     "S3 Put Object",
				Priority: 10,
				Request: models.Request{
					URI:    "/.*",
					Method: "PUT",
					IsRegex: models.RegexConfig{
						URI: true,
					},
					Headers: map[string]string{
						"x-amz-content-sha256": ".*",
					},
				},
				Response: models.Response{
					StatusCode: 200,
					Headers: map[string]string{
						"x-amz-request-id":  "{{randomString 16}}",
						"x-amz-id-2":        "{{randomString 32}}",
						"ETag":              "\"{{randomString 32}}\"",
						"x-amz-server-side-encryption": "AES256",
					},
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},

		// OpenAI API Templates
		{
			ID:          "openai-chat-completion",
			Name:        "OpenAI - Chat Completion",
			Description: "Mock OpenAI chat completion",
			Category:    TemplateOpenAI,
			Tags:        []string{"openai", "chat", "post"},
			Mock: models.Mock{
				Name:     "OpenAI Chat Completion",
				Priority: 10,
				Request: models.Request{
					URI:    "/v1/chat/completions",
					Method: "POST",
				},
				Response: models.Response{
					StatusCode: 200,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
					Body: `{
						"id": "chatcmpl-{{randomString 29}}",
						"object": "chat.completion",
						"created": {{timestamp}},
						"model": "gpt-4",
						"choices": [
							{
								"index": 0,
								"message": {
									"role": "assistant",
									"content": "This is a mocked response from OpenAI API."
								},
								"finish_reason": "stop"
							}
						],
						"usage": {
							"prompt_tokens": 10,
							"completion_tokens": 20,
							"total_tokens": 30
						}
					}`,
					Template: true,
					Delay:    1000,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},

		// Twilio API Templates
		{
			ID:          "twilio-send-sms",
			Name:        "Twilio - Send SMS",
			Description: "Mock Twilio SMS sending",
			Category:    TemplateTwilio,
			Tags:        []string{"twilio", "sms", "post"},
			Mock: models.Mock{
				Name:     "Twilio Send SMS",
				Priority: 10,
				Request: models.Request{
					URI:    "/2010-04-01/Accounts/.*/Messages.json",
					Method: "POST",
					IsRegex: models.RegexConfig{
						URI: true,
					},
				},
				Response: models.Response{
					StatusCode: 201,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
					Body: `{
						"sid": "SM{{randomString 32}}",
						"account_sid": "AC{{randomString 32}}",
						"from": "+15551234567",
						"to": "+15559876543",
						"body": "Your message here",
						"status": "queued",
						"date_created": "{{timestamp}}",
						"date_updated": "{{timestamp}}"
					}`,
					Template: true,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},

		// Slack API Templates
		{
			ID:          "slack-post-message",
			Name:        "Slack - Post Message",
			Description: "Mock Slack message posting",
			Category:    TemplateSlack,
			Tags:        []string{"slack", "message", "post"},
			Mock: models.Mock{
				Name:     "Slack Post Message",
				Priority: 10,
				Request: models.Request{
					URI:    "/api/chat.postMessage",
					Method: "POST",
				},
				Response: models.Response{
					StatusCode: 200,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
					Body: `{
						"ok": true,
						"channel": "C1234567890",
						"ts": "{{timestamp}}",
						"message": {
							"text": "{{.Request.JSONPath "$.text"}}",
							"user": "U1234567890",
							"type": "message"
						}
					}`,
					Template: true,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},

		// Google APIs Templates
		{
			ID:          "google-oauth-token",
			Name:        "Google - OAuth Token",
			Description: "Mock Google OAuth token endpoint",
			Category:    TemplateGoogle,
			Tags:        []string{"google", "oauth", "post"},
			Mock: models.Mock{
				Name:     "Google OAuth Token",
				Priority: 10,
				Request: models.Request{
					URI:    "/token",
					Method: "POST",
				},
				Response: models.Response{
					StatusCode: 200,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
					Body: `{
						"access_token": "ya29.{{randomString 100}}",
						"expires_in": 3599,
						"token_type": "Bearer",
						"refresh_token": "1//{{randomString 100}}"
					}`,
					Template: true,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},

		// PayPal API Templates
		{
			ID:          "paypal-create-order",
			Name:        "PayPal - Create Order",
			Description: "Mock PayPal order creation",
			Category:    TemplatePayPal,
			Tags:        []string{"paypal", "order", "post"},
			Mock: models.Mock{
				Name:     "PayPal Create Order",
				Priority: 10,
				Request: models.Request{
					URI:    "/v2/checkout/orders",
					Method: "POST",
				},
				Response: models.Response{
					StatusCode: 201,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
					Body: `{
						"id": "{{randomString 17}}",
						"status": "CREATED",
						"links": [
							{
								"href": "https://api.paypal.com/v2/checkout/orders/{{randomString 17}}",
								"rel": "self",
								"method": "GET"
							}
						],
						"create_time": "{{timestamp}}"
					}`,
					Template: true,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// Register all templates
	for _, template := range templates {
		manager.mu.Lock()
		manager.templates[template.ID] = &template
		manager.mu.Unlock()
	}

	return nil
}
