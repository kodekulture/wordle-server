package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/lordvidex/errs"

	"github.com/Chat-Map/wordle-server/game"
	"github.com/Chat-Map/wordle-server/repository"
	"github.com/Chat-Map/wordle-server/service/hasher"
	"github.com/Chat-Map/wordle-server/service/random"
)

var (
	ErrNoPlayer = errs.B().Code(errs.InvalidArgument).Msg("player not provided").Err()
)

// TODO: maybe create an interface and put this struct one package down
type Service struct {
	h  hasher.Bcrypt
	r  random.RandomGen
	gr repository.Game
	pr repository.Player
	cr repository.Cache
}

func (s *Service) ComparePasswords(hash, original string) error {
	return s.h.Compare(hash, original)
}

func (s *Service) CreateToken(username string, gameID uuid.UUID) string {
	return s.r.Store(username, gameID)
}

func (s *Service) GetTokenPayload(token string) (string, uuid.UUID, bool) {
	return s.r.Get(token)
}

func (s *Service) CreatePlayer(ctx context.Context, player *game.Player) error {
	if player == nil {
		return ErrNoPlayer
	}
	var err error
	player.Password, err = s.h.Hash(player.Password)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "password processing error")
	}
	err = s.pr.Create(ctx, *player)
	if err != nil {
		return errs.WrapCode(err, errs.InvalidArgument, "error creating player")
	}
	return nil
}

func (s *Service) GetPlayer(ctx context.Context, username string) (*game.Player, error) {
	p, err := s.pr.GetByUsername(ctx, username)
	if err != nil {
		return nil, errs.WrapCode(err, errs.NotFound, "player not found")
	}
	return p, nil
}

func (s *Service) GetGame(ctx context.Context, roomID uuid.UUID) (*game.Game, error) {
	room, err := s.gr.FetchGame(ctx, roomID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.InvalidArgument, "error fetching game")
	}
	return room, nil
}

func (s *Service) GetPlayerRooms(ctx context.Context, playerID int) ([]game.Game, error) {
	rooms, err := s.gr.GetGames(ctx, playerID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.InvalidArgument, "error fetching games")
	}
	return rooms, nil
}

func (s *Service) StartGame(ctx context.Context, g *game.Game) error {
	err := s.gr.StartGame(ctx, g)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "error saving game for all players")
	}
	return nil
}

func (s *Service) FinishGame(ctx context.Context, g *game.Game) error {
	err := s.gr.FinishGame(ctx, g)
	if err != nil {
		return errs.WrapCode(err, errs.Internal, "error saving game for all players")
	}
	return nil
}

func New(appCtx context.Context, gr repository.Game, pr repository.Player, cr repository.Cache) *Service {
	return &Service{
		r:  random.New(appCtx),
		gr: gr,
		pr: pr,
		cr: cr,
	}
}
