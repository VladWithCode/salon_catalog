package routes

import (
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/forms"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
	"github.com/vladwithcode/salon_catalog/internal/templates/pages/dashboard"
)

func RegisterUserRoutes(router *customServeMux) {
	router.HandleFunc("PUT /panel/usuario", auth.ValidateAuth(UpdateUser))
	router.HandleFunc("PUT /panel/usuario/contrasena", auth.ValidateAuth(UpdateUserPassword))
}

func UpdateUser(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	user, _ := db.GetUserByID(a.ID)
	toastData := components.NewToastData("Se actualizó el perfil", components.ToastSuccess, 3000, true, false)
	formState := forms.NewUserFormState()
	formState.SetIsSubmitted()

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Error al procesar el formulario. Actualiza la página e inténtalo de nuevo."
		toastData.Type = components.ToastError
		formState.SetErrorMessage("Algo salió mal. Actualiza la página e inténtalo de nuevo.")

		comps := templ.Join(
			dashboard.Profile(user, formState),
			components.ToasterToast(toastData),
		)
		templ.RenderFragments(r.Context(), w, comps, "profile-edit-data-form", "toaster-toast")
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	formState.SetFieldState("fullname", forms.FieldState{
		Value: r.FormValue("fullname"),
	})
	formState.SetFieldState("username", forms.FieldState{
		Value: r.FormValue("username"),
	})
	formState.SetFieldState("email", forms.FieldState{
		Value: r.FormValue("email"),
	})

	if err = formState.Validate(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "El formulario contiene errores."
		toastData.Type = components.ToastError
		formState.SetErrorMessage("Algo salió mal. Actualiza la página e inténtalo de nuevo.")

		comps := templ.Join(
			dashboard.Profile(user, formState),
			components.ToasterToast(toastData),
		)
		templ.RenderFragments(r.Context(), w, comps, "profile-edit-data-form", "toaster-toast")
		log.Printf("failed to validate form: %v\n", err)
		return
	}

	user.Fullname = formState.GetFieldValue("fullname")
	user.Username = formState.GetFieldValue("username")
	user.Email = formState.GetFieldValue("email")
	err = db.UpdateUser(user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al actualizar el perfil"
		toastData.Type = components.ToastError
		formState.SetErrorMessage("Algo salió mal. Actualiza la página e inténtalo de nuevo.")

		comps := templ.Join(
			dashboard.Profile(user, formState),
			components.ToasterToast(toastData),
		)
		templ.RenderFragments(r.Context(), w, comps, "profile-edit-data-form", "toaster-toast")
		log.Printf("failed to update user: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	toastData.Message = "Perfil actualizado correctamente"
	formState.SetSuccessMessage("Perfil actualizado correctamente")

	comps := templ.Join(
		dashboard.Profile(user, formState),
		components.ToasterToast(toastData),
	)
	templ.RenderFragments(r.Context(), w, comps, "profile-data", "toaster-toast")
}

func UpdateUserPassword(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	user, _ := db.GetUserByID(a.ID)
	toastData := components.NewToastData("Se actualizó la contraseña", components.ToastSuccess, 3000, true, false)
	formState := forms.NewUserFormState()
	formState.SetIsSubmitted()

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Error al procesar el formulario. Actualiza la página e inténtalo de nuevo."
		toastData.Type = components.ToastError
		formState.SetErrorMessage("Algo salió mal. Actualiza la página e inténtalo de nuevo.")

		comps := templ.Join(
			dashboard.Profile(user, formState),
			components.ToasterToast(toastData),
		)
		templ.RenderFragments(r.Context(), w, comps, "profile-edit-pass-form", "toaster-toast")
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	formState.SetFieldState("current_passwd", forms.FieldState{
		Value: r.FormValue("current_passwd"),
	})
	formState.SetFieldState("new_passwd", forms.FieldState{
		Value: r.FormValue("new_passwd"),
	})
	formState.SetFieldState("confirm_passwd", forms.FieldState{
		Value: r.FormValue("confirm_passwd"),
	})

	if err = formState.Validate(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "El formulario contiene errores."
		toastData.Type = components.ToastError
		formState.SetErrorMessage("El formulario contiene errores.")

		comps := templ.Join(
			dashboard.Profile(&db.User{}, formState),
			components.ToasterToast(toastData),
		)
		templ.RenderFragments(r.Context(), w, comps, "profile-edit-pass-form", "toaster-toast")
		log.Printf("failed to validate form: %v\n", err)
		return
	}

	if err = user.ValidatePass(formState.GetFieldValue("current_passwd")); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "La contraseña actual es incorrecta."
		toastData.Type = components.ToastError
		formState.SetFieldError("current_passwd", "La contraseña actual es incorrecta.")

		comps := templ.Join(
			dashboard.Profile(user, formState),
			components.ToasterToast(toastData),
		)
		templ.RenderFragments(r.Context(), w, comps, "profile-edit-pass-form", "toaster-toast")
		log.Printf("failed to validate user password: %v\n", err)
		return
	}

	user.HashPass(formState.GetFieldValue("new_passwd"))
	err = db.UpdateUser(user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al actualizar la contraseña"
		toastData.Type = components.ToastError

		comps := templ.Join(
			dashboard.Profile(user, formState),
			components.ToasterToast(toastData),
		)
		templ.RenderFragments(r.Context(), w, comps, "profile-edit-pass-form", "toaster-toast")
		log.Printf("failed to update user password: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	toastData.Message = "Contraseña actualizada correctamente"
	formState.SetSuccessMessage("Contraseña actualizada correctamente")

	comps := templ.Join(
		dashboard.Profile(user, formState),
		components.ToasterToast(toastData),
	)
	templ.RenderFragments(r.Context(), w, comps, "profile-data", "toaster-toast")
}
