package user

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"CredChain_Golang/domain"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)


func TestHandler_GetUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockSvc := new(MockUserService)
	mockCredRepo := new(MockCredentialRepository)
	
	handler := NewHandler(UserHandlerParams{UserSvc: mockSvc, CredRepo: mockCredRepo})
	
	t.Run("Successfully Get Users", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/users", nil)

		users := []domain.User{
			{ID: "dummy_1", Role: domain.RoleAdmin},
		}

		mockSvc.On("GetUsers", mock.Anything).Return(users, nil).Once()

		handler.GetUsers(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "dummy_1")
		mockSvc.AssertExpectations(t)
	})
}
