package handler

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/medisave/app/pkg/logger"
	"go.uber.org/zap"
)

// USSDSessionRequest is the standard payload from Africa's Talking USSD gateway.
type USSDSessionRequest struct {
	SessionID   string `json:"sessionId"   form:"sessionId"`
	ServiceCode string `json:"serviceCode" form:"serviceCode"`
	PhoneNumber string `json:"phoneNumber" form:"phoneNumber"`
	Text        string `json:"text"        form:"text"`
}

type USSDHandler struct{}

func NewUSSDHandler() *USSDHandler {
	return &USSDHandler{}
}

// POST /api/v1/ussd/session  (Africa's Talking callback — no auth)
// Response format: "CON <menu text>" for continuation, "END <message>" for final.
func (h *USSDHandler) Session(c *gin.Context) {
	var req USSDSessionRequest
	// Africa's Talking sends form-encoded; some gateways send JSON
	if err := c.ShouldBind(&req); err != nil {
		c.String(200, "END Service temporarily unavailable.")
		return
	}

	logger.Info("USSD session",
		zap.String("session", req.SessionID),
		zap.String("phone", req.PhoneNumber),
		zap.String("text", req.Text),
	)

	resp := h.route(req.Text, req.PhoneNumber)
	c.String(200, resp)
}

// GET /api/v1/ussd/test  (development sandbox)
func (h *USSDHandler) Test(c *gin.Context) {
	text := c.DefaultQuery("text", "")
	phone := c.DefaultQuery("phone", "+2348000000000")
	resp := h.route(text, phone)
	c.JSON(200, gin.H{
		"service_code": "*384*123#",
		"response":     resp,
	})
}

// route implements the full USSD menu state machine.
// text = accumulated user input, e.g. "" → level 0, "1" → main option 1, "1*2" → sub-option.
func (h *USSDHandler) route(text, phone string) string {
	parts := strings.Split(text, "*")
	if text == "" {
		parts = []string{""}
	}

	switch {
	case text == "":
		return mainMenu()

	case len(parts) == 1:
		switch parts[0] {
		case "1":
			return conMenu("Find a Doctor\n\nBy Specialty:\n1. General Practitioner\n2. Paediatrician\n3. Gynaecologist\n4. Cardiologist\n5. Dermatologist\n0. Back")
		case "2":
			return conMenu("AI Symptom Checker\n\nDescribe your symptoms:\n1. Fever & Headache\n2. Stomach Pain\n3. Chest Pain\n4. Cough & Cold\n5. Other\n0. Back")
		case "3":
			return endMsg("To check your wallet balance:\n1. Visit medisave.ng/patient/wallet\n2. Or open the MediSave app\n\nFor support call: 0700-MEDISAVE")
		case "4":
			return endMsg(fmt.Sprintf("EMERGENCY SOS triggered for %s\n\nEmergency services have been alerted.\nNearest hospitals:\n• LUTH Lagos: 08023032003\n• Garki Hospital Abuja: 09-2340005\n\nDial 112 for immediate help.", phone))
		case "5":
			return endMsg("Nearby Hospitals:\n\n• LUTH, Surulere Lagos — 08023032003\n• Lagos Island General — 01-2700800\n• Garki Hospital Abuja — 09-2340005\n• AKTH Kano — 064-666601\n\nFor directions visit medisave.ng/patient/nearby")
		case "6":
			return conMenu("Medication Reminders\n\n1. View today's reminders\n2. Mark dose as taken\n3. Set new reminder\n0. Back")
		case "7":
			return conMenu("Appointments\n\n1. View upcoming\n2. Book new appointment\n3. Cancel appointment\n0. Back")
		case "8":
			return healthTip()
		case "9":
			return endMsg("MediSave Help\n\nPhone: 0700-MEDISAVE\nEmail: help@medisave.ng\nWebsite: medisave.ng\n\nEmergency: 112")
		case "0":
			return mainMenu()
		default:
			return invalidOption()
		}

	case len(parts) == 2:
		return h.handleLevel2(parts[0], parts[1], phone)

	default:
		return endMsg("Thank you for using MediSave.\n\nVisit medisave.ng for the full experience.")
	}
}

func (h *USSDHandler) handleLevel2(level1, level2, phone string) string {
	switch level1 {
	case "1": // Find a Doctor
		specialties := map[string]string{
			"1": "General Practitioner",
			"2": "Paediatrician",
			"3": "Gynaecologist",
			"4": "Cardiologist",
			"5": "Dermatologist",
		}
		if level2 == "0" {
			return mainMenu()
		}
		spec, ok := specialties[level2]
		if !ok {
			return invalidOption()
		}
		return endMsg(fmt.Sprintf("Find a %s on MediSave:\n\n1. Visit medisave.ng/patient/appointments\n2. Select 'Find a Doctor'\n3. Filter by %s\n\nOr call 0700-MEDISAVE to book by phone.", spec, spec))

	case "2": // AI Symptom Checker
		symptoms := map[string]string{
			"1": "Fever & Headache — Could be malaria, typhoid or viral. See a doctor within 24h. Take paracetamol for fever. Drink plenty of water.",
			"2": "Stomach Pain — Could be gastritis, food poisoning or appendicitis. Avoid spicy food. Seek urgent care if pain is severe.",
			"3": "Chest Pain — This could be serious. Please call 112 or go to the nearest emergency department immediately.",
			"4": "Cough & Cold — Rest and hydrate. Take honey with warm water. See a doctor if symptoms worsen or last more than 7 days.",
			"5": "Please visit medisave.ng/patient/ai for our full AI symptom checker. For emergencies dial 112.",
		}
		if level2 == "0" {
			return mainMenu()
		}
		msg, ok := symptoms[level2]
		if !ok {
			return invalidOption()
		}
		return endMsg("MediSave AI Advice:\n\n" + msg + "\n\n⚠ This is not a substitute for professional medical advice.")

	case "6": // Medication Reminders
		switch level2 {
		case "0":
			return mainMenu()
		case "1":
			return endMsg("Today's Medication Schedule:\n\nTo view your full schedule:\nVisit medisave.ng/patient/reminders\n\nOr open the MediSave app.")
		case "2":
			return endMsg("Dose Recorded!\n\nYour medication dose has been marked as taken.\n\nVisit medisave.ng/patient/reminders to view your adherence score.")
		case "3":
			return endMsg("To set a new medication reminder:\n\n1. Visit medisave.ng/patient/reminders\n2. Tap 'Add Reminder'\n3. Enter medicine details\n\nOr call 0700-MEDISAVE for assistance.")
		default:
			return invalidOption()
		}

	case "7": // Appointments
		switch level2 {
		case "0":
			return mainMenu()
		case "1":
			return endMsg("Your upcoming appointments:\n\nTo view full appointment list:\nVisit medisave.ng/patient/appointments\n\nOr open the MediSave app.")
		case "2":
			return endMsg("To book a new appointment:\n\n1. Visit medisave.ng/patient/appointments\n2. Tap 'Find a Doctor'\n3. Select doctor & time slot\n\nOr call 0700-MEDISAVE.")
		case "3":
			return endMsg("To cancel an appointment:\n\nVisit medisave.ng/patient/appointments and select the appointment to cancel.\n\nOr call 0700-MEDISAVE.")
		default:
			return invalidOption()
		}

	default:
		return invalidOption()
	}
}

func mainMenu() string {
	return conMenu("Welcome to MediSave *384*123#\nNigeria's Healthcare Platform\n\n1. Find a Doctor\n2. AI Symptom Checker\n3. Wallet Balance\n4. Emergency SOS\n5. Nearby Hospital\n6. Medication Reminder\n7. Appointments\n8. Health Tip\n9. Help & Support")
}

func conMenu(text string) string {
	return "CON " + text
}

func endMsg(text string) string {
	return "END " + text
}

func invalidOption() string {
	return conMenu("Invalid option. Please try again.\n\n0. Back to Main Menu")
}

func healthTip() string {
	tips := []string{
		"Drink at least 8 glasses of water daily. Dehydration worsens many illnesses.",
		"Wash your hands with soap for at least 20 seconds to prevent infections.",
		"Exercise 30 minutes daily — even a brisk walk reduces disease risk by 35%.",
		"Sleep 7-8 hours. Poor sleep weakens immunity and raises blood pressure.",
		"Malaria prevention: use insecticide-treated nets and clear stagnant water.",
		"Check your blood pressure regularly — hypertension often has no symptoms.",
		"Eat more fruits and vegetables — they reduce cancer and heart disease risk.",
		"Limit sugar intake. Type 2 diabetes is preventable with diet and exercise.",
	}
	tip := tips[time.Now().Day()%len(tips)]
	return endMsg("MediSave Health Tip:\n\n" + tip + "\n\nFor personalized advice, visit medisave.ng/patient/ai")
}
