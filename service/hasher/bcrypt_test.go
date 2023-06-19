package hasher

import "testing"

func TestBcrypt(t *testing.T) {
	var bc Bcrypt

	tests := []struct {
		name            string
		password        string
		comparePassword string
		equal           bool
	}{
		{
			name:            "similiar passwords",
			password:        "test1",
			comparePassword: "test1",
			equal:           true,
		},
		{
			name:            "different passwords",
			password:        "test1",
			comparePassword: "test2",
			equal:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := bc.Hash(tt.password)
			if err != nil {
				t.Errorf("error hashing password: %v", err)
			}
			if err = bc.Compare(hash, tt.comparePassword); err != nil {
				if tt.equal {
					t.Errorf("error comparing password: %v", err)
				}
			}
		})
	}
}
