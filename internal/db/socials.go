package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrSocialLinkInsert        = errors.New("failed to insert social link")
	ErrSocialSectionInsert     = errors.New("failed to insert social section")
	ErrSocialSectionLinkInsert = errors.New("failed to insert social section link")
)

type SocialLink struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Link      string    `json:"link"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SocialSection struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SocialSectionLink is the data that may be used at any section with social links
// to display a name | label | icon | link to any social media of the company.
type SocialSectionLink struct {
	LinkID      string    `json:"link_id"`
	LinkName    string    `json:"link_name"`
	LinkURL     string    `json:"link_url"`
	SectionID   string    `json:"section_id"`
	SectionName string    `json:"section_name"`
	IconID      string    `json:"icon_id"`
	IconName    string    `json:"icon_name"`
	IconURL     string    `json:"icon_url"`
	IconType    string    `json:"icon_type"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (s SocialSectionLink) GetID() string {
	return fmt.Sprintf("%s-%s", s.LinkID, s.SectionID)
}

type SocialSectionLinkFilter struct {
	Name string
}

type SocialSectionLinkResult struct {
	SocialSectionLink []SocialSectionLink
}

// CRUD operations for SocialLink
func CreateSocialLink(link *SocialLink) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	id, err := uuid.NewV7()
	if err != nil {
		return ErrUUIDFail
	}
	link.ID = id.String()

	args := pgx.NamedArgs{
		"id":   link.ID,
		"name": link.Name,
		"link": link.Link,
	}
	_, err = conn.Exec(
		ctx,
		`INSERT INTO social_links (id, name, link) VALUES (@id, @name, @link)`,
		args,
	)
	if err != nil {
		return errors.Join(ErrSocialLinkInsert, err)
	}

	return nil
}

func FindSocialLinkByID(id string) (*SocialLink, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var link SocialLink
	err = conn.QueryRow(
		ctx,
		`SELECT id, name, link, created_at, updated_at FROM social_links WHERE id = $1`,
		id,
	).Scan(&link.ID, &link.Name, &link.Link, &link.CreatedAt, &link.UpdatedAt)

	if err != nil {
		return nil, err
	}
	return &link, nil
}

func GetSocialLinks() ([]SocialLink, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	rows, err := conn.Query(
		ctx,
		`SELECT id, name, link, created_at, updated_at FROM social_links ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []SocialLink
	for rows.Next() {
		var link SocialLink
		err = rows.Scan(&link.ID, &link.Name, &link.Link, &link.CreatedAt, &link.UpdatedAt)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	return links, nil
}

func UpdateSocialLink(link *SocialLink) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	args := pgx.NamedArgs{
		"id":   link.ID,
		"name": link.Name,
		"link": link.Link,
	}
	_, err = conn.Exec(
		ctx,
		`UPDATE social_links SET name = @name, link = @link WHERE id = @id`,
		args,
	)
	return err
}

func DeleteSocialLink(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, `DELETE FROM social_links WHERE id = $1`, id)
	return err
}

// CRUD operations for SocialSection
func CreateSocialSection(section *SocialSection) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	id, err := uuid.NewV7()
	if err != nil {
		return ErrUUIDFail
	}
	section.ID = id.String()

	args := pgx.NamedArgs{
		"id":   section.ID,
		"name": section.Name,
	}
	_, err = conn.Exec(
		ctx,
		`INSERT INTO social_sections (id, name) VALUES (@id, @name)`,
		args,
	)
	if err != nil {
		return errors.Join(ErrSocialSectionInsert, err)
	}

	return nil
}

func GetSocialSections() ([]SocialSection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	rows, err := conn.Query(
		ctx,
		`SELECT id, name, created_at, updated_at FROM social_sections ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sections []SocialSection
	for rows.Next() {
		var section SocialSection
		err = rows.Scan(&section.ID, &section.Name, &section.CreatedAt, &section.UpdatedAt)
		if err != nil {
			return nil, err
		}
		sections = append(sections, section)
	}

	return sections, nil
}

// Main function for filtering social section links
func FilterSocialSectionLinks(filters *SocialSectionLinkFilter) (*SocialSectionLinkResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	query := `
		SELECT 
			sl.id as link_id, sl.name as link_name, sl.link as link_url,
			ss.id as section_id, ss.name as section_name,
			img.id as icon_id, img.name as icon_name, img.filename as icon_url, img.file_type as icon_type,
			sls.created_at, sls.updated_at
		FROM social_links_sections sls
		JOIN social_links sl ON sls.link_id = sl.id
		JOIN social_sections ss ON sls.section_id = ss.id
		JOIN images img ON sls.icon_id = img.id`

	args := pgx.NamedArgs{}
	conditions := []string{}

	if filters != nil && filters.Name != "" {
		conditions = append(conditions, "ss.name = @section_name")
		args["section_name"] = filters.Name
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY ss.name, sl.name"

	rows, err := conn.Query(ctx, query, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []SocialSectionLink
	for rows.Next() {
		var link SocialSectionLink
		err = rows.Scan(
			&link.LinkID, &link.LinkName, &link.LinkURL,
			&link.SectionID, &link.SectionName,
			&link.IconID, &link.IconName, &link.IconURL, &link.IconType,
			&link.CreatedAt, &link.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	return &SocialSectionLinkResult{
		SocialSectionLink: links,
	}, nil
}

// Create association between social link and section
func CreateSocialSectionLink(linkID, sectionID, iconID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	args := pgx.NamedArgs{
		"link_id":    linkID,
		"section_id": sectionID,
		"icon_id":    iconID,
	}
	_, err = conn.Exec(
		ctx,
		`INSERT INTO social_links_sections (link_id, section_id, icon_id) VALUES (@link_id, @section_id, @icon_id)`,
		args,
	)
	if err != nil {
		return errors.Join(ErrSocialSectionLinkInsert, err)
	}

	return nil
}

func DeleteSocialSectionLink(linkID, sectionID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := GetConn()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(
		ctx,
		`DELETE FROM social_links_sections WHERE link_id = $1 AND section_id = $2`,
		linkID, sectionID,
	)
	return err
}
