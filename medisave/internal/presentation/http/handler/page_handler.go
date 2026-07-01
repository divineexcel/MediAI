package handler

import "github.com/gin-gonic/gin"

type PageHandler struct{}

func NewPageHandler() *PageHandler { return &PageHandler{} }

func (h *PageHandler) Login(c *gin.Context)             { c.File("./web/pages/auth/login.html") }
func (h *PageHandler) Register(c *gin.Context)          { c.File("./web/pages/auth/register.html") }
func (h *PageHandler) PatientDashboard(c *gin.Context)  { c.File("./web/pages/patient/dashboard.html") }
func (h *PageHandler) PatientProfile(c *gin.Context)    { c.File("./web/pages/patient/profile.html") }
func (h *PageHandler) PatientWallet(c *gin.Context)     { c.File("./web/pages/patient/wallet.html") }
func (h *PageHandler) PatientAI(c *gin.Context)         { c.File("./web/pages/patient/ai.html") }
func (h *PageHandler) PatientRecords(c *gin.Context)    { c.File("./web/pages/patient/records.html") }
func (h *PageHandler) PatientNearby(c *gin.Context)     { c.File("./web/pages/patient/nearby.html") }
func (h *PageHandler) PatientEmergency(c *gin.Context)  { c.File("./web/pages/patient/emergency.html") }
func (h *PageHandler) PatientReminders(c *gin.Context)  { c.File("./web/pages/patient/reminders.html") }
func (h *PageHandler) PatientSavings(c *gin.Context)    { c.File("./web/pages/patient/savings.html") }
func (h *PageHandler) PatientAppointments(c *gin.Context) { c.File("./web/pages/patient/appointments.html") }
func (h *PageHandler) DoctorDashboard(c *gin.Context)    { c.File("./web/pages/doctor/dashboard.html") }
func (h *PageHandler) DoctorProfile(c *gin.Context)      { c.File("./web/pages/doctor/profile.html") }
func (h *PageHandler) DoctorAppointments(c *gin.Context) { c.File("./web/pages/doctor/appointments.html") }
func (h *PageHandler) DoctorPatients(c *gin.Context)     { c.File("./web/pages/doctor/patients.html") }
func (h *PageHandler) DoctorEarnings(c *gin.Context)     { c.File("./web/pages/doctor/earnings.html") }
func (h *PageHandler) AdminDashboard(c *gin.Context)      { c.File("./web/pages/admin/dashboard.html") }
func (h *PageHandler) ConsultationCall(c *gin.Context)    { c.File("./web/pages/consultation/call.html") }
func (h *PageHandler) Root(c *gin.Context)               { c.Redirect(302, "/login") }
