package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/config"
	"github.com/sambit24795/cinedeck/internal/library"
	"github.com/spf13/cobra"
)

// openLibrary opens the library database at config.LibraryPath().
func openLibrary() (*library.Library, error) {
	path, err := config.LibraryPath()
	if err != nil {
		return nil, err
	}
	return library.Open(path)
}

// openLibraryIfExists opens the library at config.LibraryPath() without
// creating it. It returns (nil, nil) when the database does not exist yet.
func openLibraryIfExists() (*library.Library, error) {
	path, err := config.LibraryPath()
	if err != nil {
		return nil, err
	}
	lib, err := library.OpenExisting(path)
	if errors.Is(err, library.ErrNoDatabase) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return lib, nil
}

// itemLabel renders m as "Inception (2010) [movie:27205]"; with Year 0 it is "Title [movie:1]".
func itemLabel(m *catalog.Meta) string {
	if m.Year > 0 {
		return fmt.Sprintf("%s (%d) [%s]", m.Title, m.Year, m.ID)
	}
	return fmt.Sprintf("%s [%s]", m.Title, m.ID)
}

// lookup returns the stored title, year and label of id. When the item is not
// in the library (or cannot be read) title is "", year is 0 and label is id.String().
func lookup(ctx context.Context, lib *library.Library, id catalog.ID) (title string, year int, label string) {
	if lib == nil {
		return "", 0, id.String()
	}
	m, err := lib.Item(ctx, id)
	if err != nil {
		return "", 0, id.String()
	}
	return m.Title, m.Year, itemLabel(m)
}

// resolveRef turns arg into an id. A typed id ("movie:27205") needs no token
// and makes no request; a title uses newCatalog and resolveID.
func resolveRef(cmd *cobra.Command, arg string, first bool) (catalog.ID, error) {
	if id, err := catalog.ParseID(arg); err == nil {
		return id, nil
	}
	c, _, err := newCatalog(cmd)
	if err != nil {
		return catalog.ID{}, err
	}
	return resolveID(cmd, c, arg, first)
}

// resolveMeta resolves arg like resolveRef (but always through newCatalog, so
// a missing token is an error) and fetches the item's catalog.Meta.
func resolveMeta(cmd *cobra.Command, arg string, first bool) (*catalog.Meta, error) {
	c, _, err := newCatalog(cmd)
	if err != nil {
		return nil, err
	}
	id, err := resolveID(cmd, c, arg, first)
	if err != nil {
		return nil, err
	}
	return c.Meta(cmd.Context(), id)
}

// actionOutput is the --json result of the commands that change the library.
type actionOutput struct {
	ID      catalog.ID `json:"id"`
	Title   string     `json:"title"`
	Year    int        `json:"year"`
	Action  string     `json:"action"`
	Changed bool       `json:"changed"`
	Rating  int        `json:"rating,omitempty"`
}

// printAction writes out as JSON when --json is set, else text and a newline, to cmd.OutOrStdout().
func printAction(cmd *cobra.Command, out actionOutput, text string) error {
	if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
		return printJSON(cmd.OutOrStdout(), out)
	}
	fmt.Fprintln(cmd.OutOrStdout(), text)
	return nil
}
