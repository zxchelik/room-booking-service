package application

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zxchelik/room-booking-service/internal/handler"
	"github.com/zxchelik/room-booking-service/internal/repository"
	"github.com/zxchelik/room-booking-service/internal/service"
)

type App struct {
	Config Config
	DB     *pgxpool.Pool
	Router http.Handler
}

func New(ctx context.Context, cfg Config) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.Ping(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	if cfg.AutoMigrate {
		if err := RunMigrations(ctx, cfg.DatabaseURL); err != nil {
			db.Close()
			return nil, fmt.Errorf("run migrations: %w", err)
		}
	}

	userRepo := repository.NewUserRepository()
	roomRepo := repository.NewRoomRepository()
	scheduleRepo := repository.NewScheduleRepository()
	slotRepo := repository.NewSlotRepository()
	bookingRepo := repository.NewBookingRepository()

	authService := service.NewAuthService(db, userRepo, cfg.JWTSecret, cfg.JWTTTL, cfg.adminRegistrationEnabled())
	roomService := service.NewRoomService(db, roomRepo)
	scheduleService := service.NewScheduleService(db, roomRepo, scheduleRepo, slotRepo, cfg.SlotWindowDays)
	slotService := service.NewSlotService(db, roomRepo, scheduleRepo, slotRepo, scheduleService)
	bookingService := service.NewBookingService(db, bookingRepo, slotRepo)

	router := handler.NewRouter(handler.Services{
		Auth:      authService,
		Rooms:     roomService,
		Schedules: scheduleService,
		Slots:     slotService,
		Bookings:  bookingService,
	}, cfg.dummyLoginEnabled())

	return &App{
		Config: cfg,
		DB:     db,
		Router: router,
	}, nil
}

func (a *App) Close() {
	if a.DB != nil {
		a.DB.Close()
	}
}
