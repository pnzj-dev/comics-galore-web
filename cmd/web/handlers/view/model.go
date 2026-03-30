package view

type Tag struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
}

type Category struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
}

type SubscriptionPlan struct {
	PlanID   int     `json:"planId"`
	Name     string  `json:"name"`
	Price    float32 `json:"price"`
	Duration int     `json:"duration"`
	Discount float32 `json:"discount"`
}

type AppContext struct {
	Title             string              `json:"title,omitempty"`
	TurnstileSiteKey  string              `json:"turnstileSiteKey,omitempty"`
	TurnstileEnabled  bool                `json:"turnstileEnabled,omitempty"`
	UserInfo          *ComicsGaloreClaims `json:"userInfo,omitempty"`
	Variants          map[string]string   `json:"variants,omitempty"`
	Tags              []Tag               `json:"tags"`
	Categories        []Category          `json:"categories"`
	SubscriptionPlans []SubscriptionPlan  `json:"subscriptionPlans"`
}
