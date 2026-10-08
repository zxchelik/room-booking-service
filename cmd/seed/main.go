package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zxchelik/room-booking-service/internal/application"
	"github.com/zxchelik/room-booking-service/internal/model"
	"github.com/zxchelik/room-booking-service/internal/repository"
	"github.com/zxchelik/room-booking-service/internal/service"
)

type seedRoom struct {
	Name        string
	Description string
	Capacity    int
	DaysOfWeek  []int
	StartTime   string
	EndTime     string
}

func main() {
	cfg := application.LoadConfig()
	if cfg.AppEnv == "production" {
		log.Fatal("seed is available only in development and test environments")
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("validate config: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := application.RunMigrations(ctx, cfg.DatabaseURL); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	db, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepository()
	roomRepo := repository.NewRoomRepository()
	scheduleRepo := repository.NewScheduleRepository()
	slotRepo := repository.NewSlotRepository()

	authService := service.NewAuthService(db, userRepo, cfg.JWTSecret, cfg.JWTTTL, true)
	roomService := service.NewRoomService(db, roomRepo)
	scheduleService := service.NewScheduleService(db, roomRepo, scheduleRepo, slotRepo, cfg.SlotWindowDays)

	if err := ensureUser(ctx, userRepo, authService, db, "seed-admin@example.com", "seed-admin-password", model.RoleAdmin); err != nil {
		log.Fatalf("seed admin user: %v", err)
	}
	if err := ensureUser(ctx, userRepo, authService, db, "seed-user@example.com", "seed-user-password", model.RoleUser); err != nil {
		log.Fatalf("seed regular user: %v", err)
	}

	rooms, err := roomService.List(ctx)
	if err != nil {
		log.Fatalf("list rooms: %v", err)
	}
	roomByName := make(map[string]model.Room, len(rooms))
	for _, room := range rooms {
		roomByName[room.Name] = room
	}

	for _, seed := range []seedRoom{
		{Name: "Mercury", Description: "Small focus room", Capacity: 4, DaysOfWeek: []int{1, 2, 3, 4, 5}, StartTime: "09:00", EndTime: "18:00"},
		{Name: "Venus", Description: "Team sync room", Capacity: 8, DaysOfWeek: []int{1, 2, 3, 4, 5}, StartTime: "10:00", EndTime: "19:00"},
		{Name: "Mars", Description: "Workshop room", Capacity: 12, DaysOfWeek: []int{1, 2, 3, 4, 5}, StartTime: "08:30", EndTime: "17:00"},
	} {
		room, exists := roomByName[seed.Name]
		if !exists {
			description := seed.Description
			capacity := seed.Capacity
			room, err = roomService.Create(ctx, model.Room{
				Name:        seed.Name,
				Description: &description,
				Capacity:    &capacity,
			})
			if err != nil {
				log.Fatalf("create room %s: %v", seed.Name, err)
			}
			roomByName[seed.Name] = room
		}

		if _, err := scheduleRepo.GetByRoomID(ctx, db, room.ID); err != nil {
			if !errors.Is(err, repository.ErrNotFound) {
				log.Fatalf("get schedule for room %s: %v", seed.Name, err)
			}
			if _, err := scheduleService.Create(ctx, room.ID, model.Schedule{
				RoomID:     room.ID,
				DaysOfWeek: seed.DaysOfWeek,
				StartTime:  seed.StartTime,
				EndTime:    seed.EndTime,
			}); err != nil {
				log.Fatalf("create schedule for room %s: %v", seed.Name, err)
			}
		}
	}

	fmt.Println("seed completed")
	fmt.Println("admin: seed-admin@example.com / seed-admin-password")
	fmt.Println("user:  seed-user@example.com / seed-user-password")
}

func ensureUser(ctx context.Context, users *repository.UserRepository, auth *service.AuthService, db repository.DBTX, email, password string, role model.Role) error {
	if _, err := users.GetByEmail(ctx, db, email); err == nil {
		return nil
	} else if !errors.Is(err, repository.ErrNotFound) {
		return err
	}

	_, err := auth.Register(ctx, email, password, role)
	return err
}
