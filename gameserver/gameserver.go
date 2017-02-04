package main

import (
	"log"
	"net"
	"time"
	"math/rand"
	"database/sql"
	
        "golang.org/x/net/context"
	"golang.org/x/crypto/bcrypt"

        "google.golang.org/grpc"
        "google.golang.org/grpc/codes"
        "google.golang.org/grpc/reflection"

        pb "../proto"
)

import _ "github.com/go-sql-driver/mysql"

const (
        port = ":50052"
)

type server struct {
	db *sql.DB
}


func (s *server) CreateUser(ctx context.Context, in *pb.CreateUserRequest) (*pb.CreateUserReply, error) {

	log.Printf("Create user request for %s", in.Name)

	// create a hashed password to store in the DB
	passbytes := []byte(in.Password)
	hashedPassword, err := bcrypt.GenerateFromPassword(passbytes, bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("bcrypt failure %v", err)
	}

	// add the user to the DB
	r, err := s.db.Exec(
		"INSERT INTO Players (name, email, phone, passhash) VALUES (?, ?, ?, ?)",
		in.Name, in.Email, in.Phone, string(hashedPassword))
	if err != nil {
		log.Fatal(" CreateUser: Couldn't insert into table: %v", err)
	}
	playerID, err := r.LastInsertId()

	// create a new session for the freshly created user, add to DB
	sessID := rand.Int63()
	log.Printf(" CreateUser: creating new session %v for new user", sessID)
	_, err = s.db.Exec(
		"INSERT INTO Logins (playerID, sessionID) VALUES (?, ?)",
		playerID, sessID)
	if err != nil {
		log.Fatal(" CreateUser: Couldn't create login session: %v", err)
	}

	// return session to caller
	return &pb.CreateUserReply{SessionID: sessID}, nil
}


func (s *server) DeleteUser(ctx context.Context, in *pb.DeleteUserRequest) (*pb.DeleteUserReply, error) {

	log.Printf("Delete user request for session %v", in.SessionID)

	// does the session exist?
	var playerID int64
	err := s.db.QueryRow("SELECT playerID FROM Logins WHERE sessionID=?", in.SessionID).Scan(&playerID)
	switch {
	case err == sql.ErrNoRows:
		// TODO - note failed session?
		log.Printf(" No such session: %v", err)
		return nil, grpc.Errorf(codes.PermissionDenied, "No such session.")

	case err != nil:
		log.Fatal(" Error fetching session: %v", err)

	default:
		log.Printf(" Found player %v", playerID)
	}

	// now fetch the user's hashed password
	var passHash []byte
	err = s.db.QueryRow("SELECT passhash FROM Players WHERE playerID=?", playerID).Scan(&passHash)
	switch {
	case err == sql.ErrNoRows:
		log.Fatal(" No such user?? (surprising) %v", err)
	case err != nil:
		log.Fatal(" Error fetching user: %v", err)
	default:
	}

	givenPassword := []byte(in.Password)
	err = bcrypt.CompareHashAndPassword(passHash, givenPassword)
	if err != nil {
		// TODO - note failed attempt for security log?
		log.Printf(" Given password %v doesn't match! Won't delete %v.", string(givenPassword), err)
		return nil, grpc.Errorf(codes.PermissionDenied, "Password didn't match")
	}

	// if they made it this far, looks like we should actually delete them!

	// nuke/invalidate all login sessions
	s.db.Exec("DELETE * FROM Logins WHERE playerID=?", playerID)

	// overwrite but don't delete the playerID.
	s.db.Exec("UPDATE Players SET name='X', email='X', phone='X', passhash='X', t=NOW(), isDeleted=TRUE WHERE playerID=?", playerID)

	// return session to caller
	return &pb.DeleteUserReply{}, nil
}



// dummy Login implementation
func (s *server) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginReply, error) {
	log.Printf("Login request")
	return &pb.LoginReply{}, nil
}

// dummy ShowGames implementation
func (s *server) ShowGames(ctx context.Context, in *pb.ShowGamesRequest) (*pb.ShowGamesReply, error) {
	log.Printf("ShowGames request")
	return &pb.ShowGamesReply{}, nil
}

// dummy GetGame implementation
func (s *server) GetGame(ctx context.Context, in *pb.GetGameRequest) (*pb.GetGameReply, error) {
	log.Printf("GetGame request")
	return &pb.GetGameReply{}, nil
}

// dummy MakeMove implementation
func (s *server) MakeMove(ctx context.Context, in *pb.MakeMoveRequest) (*pb.MakeMoveReply, error) {
	log.Printf("MakeMove request")
	return &pb.MakeMoveReply{}, nil
}



func main() {
	// seed random number generator
	rand.Seed(time.Now().UTC().UnixNano())

	log.Print("Game server binding...")
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Print("Connecting to DB...")
	db, err := sql.Open("mysql", "golang:8p2!xmTQZ$@/othello")
	if err != nil {
		log.Fatalf("failed to connect to DB: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("DB connection error: %v", err)
	}
	_, err = db.Exec("INSERT INTO ServerStarts (ts) VALUES (NOW())")
	if err != nil {
		log.Fatalf("failed to update ServerStarts: %v", err)
	}
	os := server{db: db}

	log.Print("Game server listening...")
	gs := grpc.NewServer()
	pb.RegisterOthelloGameServer(gs, &os)
	reflection.Register(gs)
	if err := gs.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	log.Print("Shutting down...")
}
