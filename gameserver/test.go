package main

// This is an end-to-end test of the backend server via the gRPC API

import (
	"log"
	
        "golang.org/x/net/context"
        "google.golang.org/grpc"
        "google.golang.org/grpc/codes"
        pb "../proto"
)

const (
	address = "localhost:50052"
	defaultName = "world"
)

func main() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewOthelloGameClient(conn)

	r, err := c.CreateUser(context.Background(), &pb.CreateUserRequest{Name: "David", Email: "david@weekly.org", Phone: "415-336-2617", Password: "tr33tr33"})
	if err != nil {
		log.Fatalf("could not create user: %v", err)
	}
	sessID := r.SessionID
	log.Printf("Created user, session is now %v", r.SessionID)
	
	// now try to delete user with wrong password.
	_, err = c.DeleteUser(context.Background(), &pb.DeleteUserRequest{SessionID: sessID, Password: "f00f00"})
	if err == nil || grpc.Code(err) != codes.PermissionDenied {
		log.Fatalf("Should have been denied, instead got %v", err)
	}
	log.Printf("Good, failed to delete user with bad pass")

	// now try to delete user with wrong session.
	_, err = c.DeleteUser(context.Background(), &pb.DeleteUserRequest{SessionID: 12345, Password: "tr33tr33"})
	if err == nil || grpc.Code(err) != codes.PermissionDenied {
		log.Fatalf("Should have been denied, instead got %v", err)
	}
	log.Printf("Good, failed to delete user with bad sess")

	// now try to delete user correctly
	_, err = c.DeleteUser(context.Background(), &pb.DeleteUserRequest{SessionID: sessID, Password: "tr33tr33"})
	if err != nil {
		log.Fatalf("Should have been allowed to delete, instead got %v", err)
	}
	log.Printf("Test user deleted.")
}
