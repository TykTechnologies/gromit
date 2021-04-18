package server

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/TykTechnologies/gromit/devenv"
	"github.com/TykTechnologies/gromit/util"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// ReST API for /env

func (a *App) createEnv(w http.ResponseWriter, r *http.Request) {
	util.StatCount("env.create.count", 1)
	vars := mux.Vars(r)
	ref := vars["name"]

	newEnv := make(devenv.VersionMap)
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&newEnv)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	}

	log.Debug().Str("envname", ref).Interface("payload", newEnv).Msg("new env received")
	de := devenv.NewDevEnv(ref, a.DB, a.TableName)
	if err != nil {
		util.StatCount("env.create.failures", 1)
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	de.SetVersions(newEnv)
	err = de.Save()
	if err != nil {
		log.Error().Err(err).Str("env", ref).Msg("could not save env")
		respondWithError(w, http.StatusInternalServerError, "could not create: "+ref)
	}
	log.Info().Interface("env", de.VersionMap()).Msg("created")
	respondWithJSON(w, http.StatusCreated, de.VersionMap())
}

func (a *App) updateEnv(w http.ResponseWriter, r *http.Request) {
	util.StatCount("env.update.count", 1)
	vars := mux.Vars(r)
	ref := vars["name"]
	log.Debug().Interface("env", ref).Interface("payload", vars).Msgf("update received")

	newEnv := make(devenv.VersionMap)
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&newEnv)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
	}

	de, err := devenv.GetDevEnv(a.DB, a.TableName, ref)
	if err != nil {
		util.StatCount("update.failures", 1)
		if _, ok := err.(devenv.NotFoundError); ok {
			log.Debug().Str("env", ref).Msg("not found")
			respondWithError(w, http.StatusNotFound, "could not find env "+ref)
			return
		}
		log.Error().Err(err).Str("env", ref).Msg("could not lookup env")
		respondWithError(w, http.StatusInternalServerError, "unknown error while looking up "+ref)
		return
	}
	de.MarkNew()
	err = de.MergeVersions(newEnv)
	if err != nil {
		log.Error().Err(err).Str("env", ref).Msg("could not save env")
		respondWithError(w, http.StatusInternalServerError, "could not update: "+ref)
		return
	}
	log.Info().Interface("env", ref).Msg("env updated")
	respondWithJSON(w, http.StatusOK, de.VersionMap())
}

func (a *App) getEnv(w http.ResponseWriter, r *http.Request) {
	util.StatCount("env.get.count", 1)
	vars := mux.Vars(r)
	ref := vars["name"]
	log.Trace().Interface("env", ref).Interface("payload", vars).Msgf("get env received")

	de, err := devenv.GetDevEnv(a.DB, a.TableName, ref)
	if err != nil {
		util.StatCount("update.failures", 1)
		if _, ok := err.(devenv.NotFoundError); ok {
			log.Debug().Str("env", ref).Msg("not found")
			respondWithError(w, http.StatusNotFound, "could not find env "+ref)
			return
		}
		log.Error().Err(err).Str("env", ref).Msg("could not lookup env")
		respondWithError(w, http.StatusInternalServerError, "unknown error while looking up "+ref)
		return
	}
	log.Debug().Interface("env", ref).Msg("env found")
	respondWithJSON(w, http.StatusOK, de.VersionMap())
}

func (a *App) deleteEnv(w http.ResponseWriter, r *http.Request) {
	util.StatCount("env.delete.count", 1)
	vars := mux.Vars(r)
	ref := vars["name"]
	log.Debug().Interface("env", ref).Interface("payload", vars).Msgf("new env received")

	de, err := devenv.GetDevEnv(a.DB, a.TableName, ref)
	if err != nil {
		util.StatCount("update.failures", 1)
		if _, ok := err.(devenv.NotFoundError); ok {
			log.Debug().Str("env", ref).Msg("not found")
			respondWithError(w, http.StatusNotFound, "could not find env "+ref)
			return
		}
		log.Error().Err(err).Str("env", ref).Msg("could not lookup env")
		respondWithError(w, http.StatusInternalServerError, "unknown error while looking up "+ref)
		return
	}
	de.MarkDeleted()
	err = de.Save()
	if err != nil {
		log.Error().Err(err).Str("env", ref).Msg("could not save env")
		respondWithError(w, http.StatusInternalServerError, "could not delete: "+ref)
		return
	}
	log.Info().Interface("env", ref).Msg("marked as deleted")
	w.WriteHeader(http.StatusAccepted)
	io.WriteString(w, "ok")
}
