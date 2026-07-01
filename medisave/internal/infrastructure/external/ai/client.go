package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// AnalysisResult is what we parse from the external AI API — or compute locally.
type AnalysisResult struct {
	Content           string  `json:"content"`
	Severity          string  `json:"severity"` // low | moderate | high
	PossibleCondition string  `json:"possible_condition"`
	RecommendedAction string  `json:"recommended_action"`
	LabTests          string  `json:"lab_tests"`
	HomeCare          string  `json:"home_care"`
	EmergencyWarning  string  `json:"emergency_warning"`
	ShowBookDoctor    bool    `json:"show_book_doctor"`
	ShowFindHospital  bool    `json:"show_find_hospital"`
	ShowEmergencySOS  bool    `json:"show_emergency_sos"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string, timeoutSeconds int) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}
}

// Analyze sends the conversation history to the external AI API.
// Falls back to the local rule-based engine if the API is not configured or unavailable.
func (c *Client) Analyze(messages []ChatMessage, userMessage string) (*AnalysisResult, error) {
	if c.baseURL != "" && c.apiKey != "" {
		result, err := c.callExternal(messages)
		if err == nil {
			return result, nil
		}
		// Fall through to local if external fails
	}
	return analyzeLocally(userMessage), nil
}

func (c *Client) callExternal(messages []ChatMessage) (*AnalysisResult, error) {
	systemPrompt := ChatMessage{
		Role: "system",
		Content: `You are MediSave AI, a Nigerian medical assistant. Analyze the patient's symptoms and respond with a JSON object only:
{
  "content": "empathetic response to the patient",
  "severity": "low|moderate|high",
  "possible_condition": "2-3 possible conditions",
  "recommended_action": "what patient should do",
  "lab_tests": "relevant tests if needed (empty if not needed)",
  "home_care": "home management tips",
  "emergency_warning": "warning signs that require immediate care (empty if not severe)",
  "show_book_doctor": true|false,
  "show_find_hospital": true|false,
  "show_emergency_sos": true|false
}
Set show_emergency_sos=true only for life-threatening symptoms. Always add disclaimer about not replacing real doctors.`,
	}

	allMessages := append([]ChatMessage{systemPrompt}, messages...)
	body, _ := json.Marshal(map[string]interface{}{
		"messages":    allMessages,
		"temperature": 0.3,
		"max_tokens":  800,
	})

	req, err := http.NewRequest("POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("AI API returned %d", resp.StatusCode)
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("empty AI response")
	}

	var result AnalysisResult
	if err := json.Unmarshal([]byte(apiResp.Choices[0].Message.Content), &result); err != nil {
		// If response isn't valid JSON, treat it as plain content
		result.Content = apiResp.Choices[0].Message.Content
		result.Severity = "low"
	}
	return &result, nil
}

// ─── Local rule-based engine ─────────────────────────────────────────────────

type rule struct {
	keywords  []string
	severity  string
	condition string
	action    string
	labTests  string
	homeCare  string
	emergency string
	bookDoc   bool
	hospital  bool
	sos       bool
}

var rules = []rule{
	{
		keywords:  []string{"chest pain", "chest tightness", "heart attack", "can't breathe", "cannot breathe", "difficulty breathing", "shortness of breath", "stroke", "unconscious", "fainted", "severe bleeding", "poison", "overdose", "seizure", "convulsion"},
		severity:  "high",
		condition: "Possible cardiac or respiratory emergency",
		action:    "Call emergency services immediately. Do not wait. Go to the nearest emergency room.",
		emergency: "This may be life-threatening. Call emergency services NOW.",
		bookDoc:   false, hospital: true, sos: true,
	},
	{
		keywords:  []string{"fever", "high temperature", "38", "39", "40", "malaria", "typhoid"},
		severity:  "moderate",
		condition: "Possible malaria, typhoid, or viral infection",
		action:    "See a doctor within 24 hours for a blood test. Take paracetamol to reduce fever.",
		labTests:  "Malaria RDT or thick film, FBC, Widal test, Blood culture",
		homeCare:  "Rest, stay hydrated with ORS or water, cool compress on forehead, paracetamol 500mg every 6 hours",
		bookDoc:   true, hospital: false, sos: false,
	},
	{
		keywords:  []string{"headache", "migraine", "head pain"},
		severity:  "low",
		condition: "Tension headache, migraine, or dehydration",
		action:    "Rest in a quiet dark room. If it persists more than 24 hours or is the worst headache of your life, see a doctor.",
		homeCare:  "Drink plenty of water, rest, paracetamol or ibuprofen as directed. Avoid screen time.",
		bookDoc:   false, hospital: false, sos: false,
	},
	{
		keywords:  []string{"stomach pain", "abdominal pain", "belly pain", "stomach ache", "diarrhea", "diarrhoea", "vomiting", "nausea", "food poisoning"},
		severity:  "moderate",
		condition: "Gastroenteritis, food poisoning, or peptic ulcer",
		action:    "Stay hydrated with ORS. If vomiting blood or pain is severe, go to hospital.",
		labTests:  "Stool microscopy and culture, H. pylori test, FBC",
		homeCare:  "Oral Rehydration Solution (ORS), avoid solid food for 4-6 hours, small sips of water, BRAT diet (bananas, rice, applesauce, toast)",
		bookDoc:   true, hospital: false, sos: false,
	},
	{
		keywords:  []string{"cough", "cold", "runny nose", "sore throat", "flu", "influenza", "sneezing"},
		severity:  "low",
		condition: "Upper respiratory tract infection (URTI) or common cold",
		action:    "Rest and stay hydrated. See a doctor if symptoms persist beyond 7 days or you develop breathing difficulty.",
		homeCare:  "Honey and warm water, steam inhalation, paracetamol for fever/pain, zinc lozenges, vitamin C",
		bookDoc:   false, hospital: false, sos: false,
	},
	{
		keywords:  []string{"rash", "skin", "itching", "itch", "hives", "allergy", "allergic"},
		severity:  "moderate",
		condition: "Allergic reaction, eczema, or contact dermatitis",
		action:    "Identify and avoid the trigger. See a dermatologist if it spreads rapidly.",
		labTests:  "Allergy panel, skin patch test",
		homeCare:  "Avoid scratching, cold compress, hydrocortisone cream (mild cases), oral antihistamine like loratadine",
		bookDoc:   true, hospital: false, sos: false,
	},
	{
		keywords:  []string{"blood pressure", "hypertension", "dizziness", "dizzy", "lightheaded"},
		severity:  "moderate",
		condition: "Hypertension or hypotension (blood pressure abnormality)",
		action:    "Check your blood pressure. If above 180/120 or below 90/60 with symptoms, seek immediate care.",
		labTests:  "Blood pressure monitoring, FBC, lipid profile, kidney function test, ECG",
		homeCare:  "Reduce salt intake, avoid stress, regular light exercise, take prescribed medications consistently",
		bookDoc:   true, hospital: false, sos: false,
	},
	{
		keywords:  []string{"diabetes", "sugar", "blood sugar", "glucose", "thirst", "frequent urination", "urinating a lot"},
		severity:  "moderate",
		condition: "Possible diabetes or blood sugar imbalance",
		action:    "Monitor blood sugar and see a doctor for proper testing.",
		labTests:  "Fasting blood glucose, HbA1c, urine dipstick, kidney function test",
		homeCare:  "Reduce sugar and refined carbs, stay active, drink water, eat small frequent meals",
		bookDoc:   true, hospital: false, sos: false,
	},
	{
		keywords:  []string{"back pain", "waist pain", "spine", "lower back"},
		severity:  "low",
		condition: "Muscle strain, lumbar disc disease, or kidney-related back pain",
		action:    "Rest, apply heat, and take anti-inflammatories. See a doctor if pain radiates to your legs or doesn't improve in 3 days.",
		homeCare:  "Warm compress, gentle stretching, ibuprofen 400mg after food, avoid heavy lifting",
		bookDoc:   false, hospital: false, sos: false,
	},
	{
		keywords:  []string{"eye", "blurry vision", "vision", "eye pain", "red eye", "discharge from eye"},
		severity:  "moderate",
		condition: "Conjunctivitis, eye infection, or refractive error",
		action:    "Avoid touching your eyes and see an ophthalmologist.",
		labTests:  "Eye examination, visual acuity test",
		homeCare:  "Clean eye with sterile saline, avoid contact lenses, do not share towels",
		bookDoc:   true, hospital: false, sos: false,
	},
	{
		keywords:  []string{"depression", "anxiety", "mental health", "panic", "stress", "sad", "hopeless", "suicidal"},
		severity:  "moderate",
		condition: "Mental health concern — depression, anxiety, or stress disorder",
		action:    "You are not alone. Please speak to a mental health professional. Your wellbeing matters.",
		homeCare:  "Talk to someone you trust, limit social media, get adequate sleep, gentle exercise, mindfulness",
		emergency: "If you are having thoughts of harming yourself, please call emergency services or a trusted person immediately.",
		bookDoc:   true, hospital: false, sos: false,
	},
}

func analyzeLocally(message string) *AnalysisResult {
	msg := strings.ToLower(message)

	for _, r := range rules {
		for _, kw := range r.keywords {
			if strings.Contains(msg, kw) {
				content := buildLocalContent(r)
				return &AnalysisResult{
					Content:           content,
					Severity:          r.severity,
					PossibleCondition: r.condition,
					RecommendedAction: r.action,
					LabTests:          r.labTests,
					HomeCare:          r.homeCare,
					EmergencyWarning:  r.emergency,
					ShowBookDoctor:    r.bookDoc,
					ShowFindHospital:  r.hospital,
					ShowEmergencySOS:  r.sos,
				}
			}
		}
	}

	// Generic fallback
	return &AnalysisResult{
		Content:           "Thank you for sharing your symptoms. I understand this can be concerning. Based on what you've described, I'd recommend monitoring your symptoms and consulting a doctor if they persist or worsen. Remember, early medical attention is always better.",
		Severity:          "low",
		PossibleCondition: "Unclear — more information needed",
		RecommendedAction: "Monitor symptoms. If they worsen or new symptoms develop, see a doctor.",
		HomeCare:          "Rest well, stay hydrated, eat balanced meals, and avoid self-medicating without medical advice.",
		ShowBookDoctor:    false,
		ShowFindHospital:  false,
		ShowEmergencySOS:  false,
	}
}

func buildLocalContent(r rule) string {
	var b strings.Builder
	switch r.severity {
	case "high":
		b.WriteString("⚠️ This sounds serious and requires immediate attention. ")
	case "moderate":
		b.WriteString("I understand you're not feeling well. Let me help you. ")
	default:
		b.WriteString("I hear you. Here's what I think may be happening. ")
	}
	if r.condition != "" {
		b.WriteString("Based on your symptoms, this could be: **" + r.condition + "**. ")
	}
	if r.action != "" {
		b.WriteString(r.action + " ")
	}
	if r.homeCare != "" && r.severity != "high" {
		b.WriteString("\n\nFor home care: " + r.homeCare + ".")
	}
	b.WriteString("\n\n*I'm an AI assistant, not a substitute for professional medical advice. Always consult a qualified doctor.*")
	return b.String()
}
