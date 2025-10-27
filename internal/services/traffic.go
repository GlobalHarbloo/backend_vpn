package services

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/yourusername/vpn-backend/internal/models"
	"github.com/yourusername/vpn-backend/internal/repository"
)

type TrafficService struct {
	UserRepo       *repository.UserRepository
	PaymentService *PaymentService
}

func NewTrafficService(userRepo *repository.UserRepository, paymentService *PaymentService) *TrafficService {
	return &TrafficService{
		UserRepo:       userRepo,
		PaymentService: paymentService,
	}
}

// StartBackgroundSync запускает фоновую задачу, которая каждые 30 секунд опрашивает Xray
// и сохраняет актуальный использованный трафик для всех пользователей в БД.
func (s *TrafficService) StartBackgroundSync() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			users, err := s.UserRepo.GetAllUsers()
			if err != nil {
				log.Printf("[Traffic] Failed to get users for sync: %v", err)
				continue
			}
			log.Printf("[Traffic] Sync cycle: found %d users", len(users))
			for _, u := range users {
				used := s.fetchTrafficForUser(&u)
				// Защита: если fetch вернул -1 — пропускаем этого пользователя
				if used < 0 {
					log.Printf("[Traffic] No stats for user %s (uuid=%s)", u.Email, u.UUID)
					continue
				}
				if err := s.UserRepo.UpdateUsedTraffic(int(u.ID), used); err != nil {
					log.Printf("[Traffic] Failed to update used traffic for %s: %v", u.UUID, err)
				} else {
					log.Printf("[Traffic] Updated used traffic for %s: old=%d new=%d", u.UUID, u.UsedTraffic, used)
				}
			}
		}
	}()
}

// fetchTrafficForUser опрашивает Xray stats по UUID и email и возвращает суммарный трафик (uplink+downlink).
// В случае ошибки возвращает -1.
func (s *TrafficService) fetchTrafficForUser(user *models.User) int64 {
	query := func(pattern string) int64 {
		req := fmt.Sprintf(`{"jsonrpc":"2.0","method":"StatsService.QueryStats","params":{"pattern":"%s","reset":false},"id":1}`, pattern)
		out, err := doPostStats(req)
		if err != nil {
			log.Printf("[Traffic] stats request failed for pattern %s: %v", pattern, err)
			return -1
		}
		v, err := parseTrafficResponse(out)
		if err != nil {
			log.Printf("[Traffic] failed to parse stats response for pattern %s: %v; raw=%s", pattern, err, string(out))
			return -1
		}
		return v
	}

	// Попробуем сначала по UUID, затем по email — возьмём максимум
	byUUID := query(fmt.Sprintf("user>>>%s>>>traffic>>>uplink", user.UUID))
	if byUUID < 0 {
		// попытка получить полные uplink+downlink
		byUUIDUp := query(fmt.Sprintf("user>>>%s>>>traffic>>>uplink", user.UUID))
		byUUIDDown := query(fmt.Sprintf("user>>>%s>>>traffic>>>downlink", user.UUID))
		if byUUIDUp < 0 || byUUIDDown < 0 {
			byUUID = -1
		} else {
			byUUID = byUUIDUp + byUUIDDown
		}
	} else {
		// получили только uplink; получим downlink
		down := query(fmt.Sprintf("user>>>%s>>>traffic>>>downlink", user.UUID))
		if down >= 0 {
			byUUID = byUUID + down
		}
	}

	byEmail := query(fmt.Sprintf("user>>>%s>>>traffic>>>uplink", user.Email))
	if byEmail < 0 {
		byEmailUp := query(fmt.Sprintf("user>>>%s>>>traffic>>>uplink", user.Email))
		byEmailDown := query(fmt.Sprintf("user>>>%s>>>traffic>>>downlink", user.Email))
		if byEmailUp < 0 || byEmailDown < 0 {
			byEmail = -1
		} else {
			byEmail = byEmailUp + byEmailDown
		}
	} else {
		down := query(fmt.Sprintf("user>>>%s>>>traffic>>>downlink", user.Email))
		if down >= 0 {
			byEmail = byEmail + down
		}
	}

	// Выбрать максимум между по email и по UUID, игнорируя -1
	if byUUID < 0 && byEmail < 0 {
		return -1
	}
	if byEmail >= byUUID {
		return byEmail
	}
	return byUUID
}

// GetUserTraffic returns the total traffic used by a user.
func (s *TrafficService) GetUserTraffic(userUUID string) (int64, error) {
	user, err := s.UserRepo.GetUserByUUID(userUUID)
	if err != nil {
		return 0, fmt.Errorf("user not found: %w", err)
	}
	// Сброс трафика раз в месяц
	monthAgo := time.Now().AddDate(0, -1, 0)
	if user.CreatedAt.Before(monthAgo) && user.UsedTraffic > 0 {
		err := s.UserRepo.UpdateUsedTraffic(int(user.ID), 0)
		if err != nil {
			log.Printf("[Traffic] Error resetting traffic for user %s: %v", userUUID, err)
		}
		return 0, nil
	}

	// Если Xray API недоступен, возвращаем сохраненный трафик из БД
	if user.UsedTraffic > 0 {
		return user.UsedTraffic, nil
	}

	getSum := func(identifier string) int64 {
		uplinkRequest := fmt.Sprintf(`{"jsonrpc":"2.0","method":"StatsService.QueryStats","params":{"pattern":"user>>>%s>>>traffic>>>uplink","reset":false},"id":1}`, identifier)
		output, err := doPostStats(uplinkRequest)
		if err != nil {
			return 0
		}
		uplinkTraffic, err := parseTrafficResponse(output)
		if err != nil {
			uplinkTraffic = 0
		}
		downlinkRequest := fmt.Sprintf(`{"jsonrpc":"2.0","method":"StatsService.QueryStats","params":{"pattern":"user>>>%s>>>traffic>>>downlink","reset":false},"id":1}`, identifier)
		output, err = doPostStats(downlinkRequest)
		if err != nil {
			return uplinkTraffic
		}
		downlinkTraffic, err := parseTrafficResponse(output)
		if err != nil {
			downlinkTraffic = 0
		}
		return uplinkTraffic + downlinkTraffic
	}

	byEmail := getSum(user.Email)
	byUUID := getSum(user.UUID)
	if byEmail >= byUUID {
		return byEmail, nil
	}
	return byUUID, nil
}

// doPostStats выполняет простой TCP POST к локальному StatsService и возвращает
// сырые байты ответа. Это позволяет корректно получать тело как в случае
// HTTP/1.x (headers+body), так и в случае HTTP/0.9 (только тело).
func doPostStats(body string) ([]byte, error) {
	addr := "127.0.0.1:10085"
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	defer conn.Close()

	// Установим дедлайн на операции записи/чтения
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		// не фатально
		log.Printf("[Traffic] SetDeadline error: %v", err)
	}

	// Формируем обычный HTTP/1.1 запрос. Если сервер отвечает в HTTP/0.9,
	// он может проигнорировать заголовки и вернуть только тело — мы всё равно
	// прочитаем байты до закрытия соединения.
	req := fmt.Sprintf("POST /stats/query HTTP/1.1\r\nHost: 127.0.0.1:10085\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s", len(body), body)
	if _, err := conn.Write([]byte(req)); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	resp, err := io.ReadAll(conn)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	return resp, nil
}

// parseTrafficResponse парсит бинарный ответ от Xray API
func parseTrafficResponse(response []byte) (int64, error) {
	// Если ответ содержит HTTP/1.x заголовки — отделим тело
	body := response
	if idx := bytes.Index(response, []byte("\r\n\r\n")); idx != -1 {
		body = response[idx+4:]
	}

	// Попытка 1: обычный JSON-ответ {"jsonrpc":...}
	type xrayStat struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			Stat struct {
				Name  string `json:"name"`
				Value int64  `json:"value"`
			} `json:"stat"`
		} `json:"result"`
	}
	var stat xrayStat
	if err := json.Unmarshal(body, &stat); err == nil {
		return stat.Result.Stat.Value, nil
	}

	// Если JSON не распарсился — попробуем вытащить JSON-подстроку (на случай управляющих символов / HTTP/0.9 обрывов)
	s := strings.TrimSpace(string(body))
	// Найдём первый '{' и последний '}' и попытаемся распарсить подстроку
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		sub := s[start : end+1]
		if err := json.Unmarshal([]byte(sub), &stat); err == nil {
			return stat.Result.Stat.Value, nil
		}
	}

	// Ещё вариант: ответ может быть просто числом или содержать число — попробуем извлечь цифры
	// Удалим все не-цифровые символы кроме минуса
	filtered := strings.Builder{}
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '-' {
			filtered.WriteRune(r)
		}
	}
	numStr := filtered.String()
	if numStr != "" {
		if v, err := strconv.ParseInt(numStr, 10, 64); err == nil {
			return v, nil
		}
	}

	// Попробуем прочитать бинарный ответ: если тело содержит бинар и >=8 байт,
	// попробуем интерпретировать первые 8 байт как uint64 (little-endian, затем big-endian).
	if len(body) >= 8 {
		le := int64(binary.LittleEndian.Uint64(body[:8]))
		if le > 0 {
			// Логируем для диагностики: какое число распознано и preview первых байт
			preview := body[:8]
			log.Printf("[Traffic] parsed binary value (LE): %d, preview=%x", le, preview)
			return le, nil
		}
		be := int64(binary.BigEndian.Uint64(body[:8]))
		if be > 0 {
			preview := body[:8]
			log.Printf("[Traffic] parsed binary value (BE): %d, preview=%x", be, preview)
			return be, nil
		}
	}

	// Фallback: вернём ошибку с hex-превью первых байт для диагностики
	previewLen := 64
	if len(body) < previewLen {
		previewLen = len(body)
	}
	return 0, fmt.Errorf("failed to parse Xray response as JSON/number/binary; preview=%x", body[:previewLen])
}

// TrackTrafficUsage records the traffic usage for a user.
func (s *TrafficService) TrackTrafficUsage(userUUID string, bytes int64) error {
	user, err := s.UserRepo.GetUserByUUID(userUUID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Обновляем использованный трафик в базе данных
	newTraffic := user.UsedTraffic + bytes
	if err := s.UserRepo.UpdateUsedTraffic(int(user.ID), newTraffic); err != nil {
		return fmt.Errorf("failed to update traffic: %w", err)
	}

	log.Printf("[Traffic] Updated traffic for user %s: +%d bytes, total: %d", userUUID, bytes, newTraffic)
	return nil
}

// CheckTrafficLimits checks if the user has exceeded the traffic limits of their current tariff.
func (s *TrafficService) CheckTrafficLimits(userID int) (bool, error) {
	// Delegate the traffic limit check to the PaymentService
	exceeded, err := s.PaymentService.CheckTariffLimits(userID)
	if err != nil {
		return false, fmt.Errorf("failed to check traffic limits: %w", err)
	}
	return exceeded, nil
}
