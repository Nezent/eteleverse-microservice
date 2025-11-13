package dto

// CreateUserRequest represents the payload for creating a new user.
type CreateUserRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// CreateUserResponse represents the response after creating a new user.
type CreateUserResponse struct {
	ID string `json:"id"`
}

// UserDetail represents the details of a user.
type UserDetail struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GetUserResponse represents the response for fetching users.
type GetUserResponse struct {
	Users []UserDetail `json:"users"`
}
