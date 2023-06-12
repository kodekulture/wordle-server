package service

import (
	"context"

	"github.com/lordvidex/errs"

	"github.com/Chat-Map/wordle-server/game"
	"github.com/Chat-Map/wordle-server/repository"
	"github.com/Chat-Map/wordle-server/service/hasher"
)

var (
	ErrNoPlayer = errs.B().Code(errs.InvalidArgument).Msg("player not provided").Err()
)

// TODO: maybe create an interface and put this struct one package down
type Service struct {
	gr repository.Game
	pr repository.Player
	h  hasher.Bcrypt
	// TODO: link hub here, so that we can add stuff to temporary area instead of repository
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

func (s *Service) GetPlayerRooms(ctx context.Context, playerID int) ([]game.Game, error) {
	rooms, err := s.gr.GetGames(ctx, playerID)
	if err != nil {
		return nil, errs.WrapCode(err, errs.InvalidArgument, "error fetching games")
	}
	return rooms, nil
}

func New(gr repository.Game, pr repository.Player) *Service {
	return &Service{
		gr: gr,
		pr: pr,
	}
}
