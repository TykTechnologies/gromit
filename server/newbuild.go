package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/TykTechnologies/gromit/devenv"
	"github.com/TykTechnologies/gromit/util"
	"github.com/rs/zerolog/log"
)

// newBuild is the handler that is invoked from github
func (a *App) newBuild(w http.ResponseWriter, r *http.Request) {
	util.StatCount("newbuild.count", 1)
	newBuild := make(map[string]string)
	err := json.NewDecoder(r.Body).Decode(&newBuild)
	if err != nil {
		util.StatCount("newbuild.failures", 1)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Trace().Interface("newBuild", newBuild).Msg("parsed from github")

	// Github sends org/reponame
	repo := getTrailingElement(newBuild["repo"], "/")
	// Github sends a path like refs/.../heads/<ref that we want>
	// Also remove all . as it will cause a problem with DNS
	ref := strings.Replace(getTrailingElement(newBuild["ref"], "/"), ".", "", -1)
	sha := newBuild["sha"]

	log.Debug().Str("repo", repo).Str("ref", ref).Str("sha", sha).Msg("to be inserted")

	de, err := devenv.GetDevEnv(a.DB, a.TableName, ref)
	if err != nil {
		if derr, ok := err.(devenv.NotFoundError); ok {
			log.Info().Str("env", ref).Msg("not found, creating")
			de = devenv.NewDevEnv(ref, a.DB, a.TableName)
		} else {
			util.StatCount("newbuild.failures", 1)
			log.Error().Err(derr).Str("env", ref).Msg("could not lookup env")
			respondWithError(w, http.StatusInternalServerError, "could not lookup env "+ref)
			return
		}
	}
	de.MarkNew()
	vs, err := devenv.GetECRState(a.ECR, a.RegistryID, ref, a.Repos)
	if err != nil {
		util.StatCount("newbuild.failures", 1)
		log.Error().Err(err).Str("env", ref).Msg("could not find ecr state")
		respondWithError(w, http.StatusInternalServerError, "could not find ecr state "+ref)
		return
	}
	de.SetVersions(vs)
	de.SetVersion(repo, sha)
	err = de.Save()
	if err != nil {
		log.Error().Err(err).Str("env", ref).Msg("could not save env")
		respondWithError(w, http.StatusInternalServerError, "could not mark as new: "+ref)
	}
	respondWithJSON(w, http.StatusOK, de.VersionMap())
}
