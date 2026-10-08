package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	handlermw "github.com/zxchelik/room-booking-service/internal/handler/middleware"
	"github.com/zxchelik/room-booking-service/internal/service"
)

type Services struct {
	Auth      *service.AuthService
	Rooms     *service.RoomService
	Schedules *service.ScheduleService
	Slots     *service.SlotService
	Bookings  *service.BookingService
}

func NewRouter(services Services, enableDummyLogin bool) http.Handler {
	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)

	authHandler := NewAuthHandler(services.Auth)
	roomHandler := NewRoomHandler(services.Rooms)
	scheduleHandler := NewScheduleHandler(services.Schedules)
	slotHandler := NewSlotHandler(services.Slots)
	bookingHandler := NewBookingHandler(services.Bookings)
	swaggerHandler := NewSwaggerHandler()

	router.Get("/_info", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	router.Get("/openapi.yaml", swaggerHandler.OpenAPISpec)
	router.Get("/swagger", swaggerHandler.UI)
	router.Post("/register", authHandler.Register)
	router.Post("/login", authHandler.Login)
	if enableDummyLogin {
		router.Post("/dummyLogin", authHandler.DummyLogin)
	}

	router.Group(func(r chi.Router) {
		r.Use(handlermw.RequireAuth(services.Auth.JWTSecret()))
		r.Get("/rooms/list", roomHandler.List)
		r.Post("/rooms/create", roomHandler.Create)
		r.Post("/rooms/{roomId}/schedule/create", scheduleHandler.Create)
		r.Get("/rooms/{roomId}/slots/list", slotHandler.List)
		r.Post("/bookings/create", bookingHandler.Create)
		r.Get("/bookings/list", bookingHandler.ListAll)
		r.Get("/bookings/my", bookingHandler.ListMy)
		r.Post("/bookings/{bookingId}/cancel", bookingHandler.Cancel)
	})

	return router
}
