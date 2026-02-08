package repository

import (
	"errors"
	"iot-backend-room-monitoring/internal/models"

	"gorm.io/gorm"
)

type RoomRepository struct {
	db *gorm.DB
}

func NewRoomRepo(db *gorm.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

// GetAllRooms retrieves all active rooms
func (r *RoomRepository) GetAllRooms() ([]models.Room, error) {
	var rooms []models.Room
	err := r.db.Where("is_active = ?", true).
		Preload("Hospital").
		Order("hospital_id ASC, room_code ASC").
		Find(&rooms).Error
	return rooms, err
}

// GetRoomByID retrieves a room by ID
func (r *RoomRepository) GetRoomByID(id uint) (*models.Room, error) {
	var room models.Room
	err := r.db.Where("id = ? AND is_active = ?", id, true).First(&room).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("room not found")
		}
		return nil, err
	}
	return &room, nil
}

// GetRoomWithHospital retrieves a room with hospital information preloaded
func (r *RoomRepository) GetRoomWithHospital(roomID uint) (*models.Room, error) {
	var room models.Room
	err := r.db.Where("id = ? AND is_active = ?", roomID, true).
		Preload("Hospital").
		First(&room).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("room not found")
		}
		return nil, err
	}
	return &room, nil
}

// GetRoomsByHospitalID retrieves all rooms for a specific hospital
func (r *RoomRepository) GetRoomsByHospitalID(hospitalID uint) ([]models.Room, error) {
	var rooms []models.Room
	err := r.db.Where("hospital_id = ? AND is_active = ?", hospitalID, true).
		Order("room_code ASC").
		Find(&rooms).Error
	return rooms, err
}

// CreateRoom creates a new room
func (r *RoomRepository) CreateRoom(room *models.Room) error {
	return r.db.Create(room).Error
}

// UpdateRoom updates an existing room
func (r *RoomRepository) UpdateRoom(room *models.Room) error {
	return r.db.Save(room).Error
}

// SoftDeleteRoom soft deletes a room by setting is_active to false
func (r *RoomRepository) SoftDeleteRoom(id uint) error {
	return r.db.Model(&models.Room{}).
		Where("id = ?", id).
		Update("is_active", false).Error
}

// GetRoomByCodeAndHospital retrieves a room by room code and hospital ID
func (r *RoomRepository) GetRoomByCodeAndHospital(roomCode string, hospitalID uint) (*models.Room, error) {
	var room models.Room
	err := r.db.Where("room_code = ? AND hospital_id = ? AND is_active = ?", roomCode, hospitalID, true).
		First(&room).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("room not found")
		}
		return nil, err
	}
	return &room, nil
}

// GetRoomsByUserID retrieves all rooms accessible by a user (via hospital access)
func (r *RoomRepository) GetRoomsByUserID(userID uint) ([]models.Room, error) {
	var rooms []models.Room
	err := r.db.
		Joins("INNER JOIN user_hospitals ON user_hospitals.hospital_id = rooms.hospital_id").
		Where("user_hospitals.user_id = ? AND rooms.is_active = ?", userID, true).
		Preload("Hospital").
		Order("rooms.hospital_id ASC, rooms.room_code ASC").
		Find(&rooms).Error
	return rooms, err
}
