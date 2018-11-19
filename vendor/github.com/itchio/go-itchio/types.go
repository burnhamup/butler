package itchio

import (
	"time"
)

// User represents an itch.io account, with basic profile info
type User struct {
	// Site-wide unique identifier generated by itch.io
	ID int64 `json:"id"`

	// The user's username (used for login)
	Username string `json:"username"`
	// The user's display name: human-friendly, may contain spaces, unicode etc.
	DisplayName string `json:"displayName"`

	// Has the user opted into creating games?
	Developer bool `json:"developer" hades:"-"`
	// Is the user part of itch.io's press program?
	PressUser bool `json:"pressUser" hades:"-"`

	// The address of the user's page on itch.io
	URL string `json:"url"`
	// User's avatar, may be a GIF
	CoverURL string `json:"coverUrl"`
	// Static version of user's avatar, only set if the main cover URL is a GIF
	StillCoverURL string `json:"stillCoverUrl"`
}

// Game represents a page on itch.io, it could be a game,
// a tool, a comic, etc.
type Game struct {
	// Site-wide unique identifier generated by itch.io
	ID int64 `json:"id"`
	// Canonical address of the game's page on itch.io
	URL string `json:"url,omitempty"`

	// Human-friendly title (may contain any character)
	Title string `json:"title,omitempty"`
	// Human-friendly short description
	ShortText string `json:"shortText,omitempty"`
	// Downloadable game, html game, etc.
	Type GameType `json:"type,omitempty"`
	// Classification: game, tool, comic, etc.
	Classification GameClassification `json:"classification,omitempty"`

	// Configuration for embedded (HTML5) games
	// @optional
	Embed *GameEmbedData `json:"embed,omitempty"`

	// Cover url (might be a GIF)
	CoverURL string `json:"coverUrl,omitempty"`
	// Non-gif cover url, only set if main cover url is a GIF
	StillCoverURL string `json:"stillCoverUrl,omitempty"`

	// Date the game was created
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	// Date the game was published, empty if not currently published
	PublishedAt *time.Time `json:"publishedAt,omitempty"`

	// Price in cents of a dollar
	MinPrice int64 `json:"minPrice,omitempty"`

	// Are payments accepted?
	CanBeBought bool `json:"canBeBought,omitempty"`

	// Does this game have a demo available?
	HasDemo bool `json:"hasDemo,omitempty"`

	// Is this game part of the itch.io press system?
	InPressSystem bool `json:"inPressSystem,omitempty"`

	// Platforms this game is available for
	Platforms Platforms `json:"platforms" hades:"squash"`

	// The user account this game is associated to
	// @optional
	User *User `json:"user,omitempty"`

	// ID of the user account this game is associated to
	UserID int64 `json:"userId,omitempty"`

	// The best current sale for this game
	// @optional
	Sale *Sale `json:"sale,omitempty"`

	// Owner-only fields

	ViewsCount     int64 `json:"viewsCount,omitempty" hades:"-"`
	DownloadsCount int64 `json:"downloadsCount,omitempty" hades:"-"`
	PurchasesCount int64 `json:"purchasesCount,omitempty" hades:"-"`

	Published bool `json:"published,omitempty" hades:"-"`
}

// Platforms describes which OS/architectures a game or upload
// is compatible with.
type Platforms struct {
	Windows Architectures `json:"windows,omitempty"`
	Linux   Architectures `json:"linux,omitempty"`
	OSX     Architectures `json:"osx,omitempty"`
}

// Architectures describes a set of processor architectures (mostly 32-bit vs 64-bit)
type Architectures string

const (
	// ArchitecturesAll represents any processor architecture
	ArchitecturesAll Architectures = "all"
	// Architectures386 represents 32-bit processor architectures
	Architectures386 Architectures = "386"
	// ArchitecturesAmd64 represents 64-bit processor architectures
	ArchitecturesAmd64 Architectures = "amd64"
)

// GameType is the type of an itch.io game page, mostly related to
// how it should be presented on web (downloadable or embed)
type GameType string

const (
	// GameTypeDefault is downloadable games
	GameTypeDefault GameType = "default"
	// GameTypeFlash is for .swf (legacy)
	GameTypeFlash GameType = "flash"
	// GameTypeUnity is for .unity3d (legacy)
	GameTypeUnity GameType = "unity"
	// GameTypeJava is for .jar (legacy)
	GameTypeJava GameType = "java"
	// GameTypeHTML is for .html (thriving)
	GameTypeHTML GameType = "html"
)

// GameClassification is the creator-picked classification for a page
type GameClassification string

const (
	// GameClassificationGame is something you can play
	GameClassificationGame GameClassification = "game"
	// GameClassificationTool includes all software pretty much
	GameClassificationTool GameClassification = "tool"
	// GameClassificationAssets includes assets: graphics, sounds, etc.
	GameClassificationAssets GameClassification = "assets"
	// GameClassificationGameMod are game mods (no link to game, purely creator tagging)
	GameClassificationGameMod GameClassification = "game_mod"
	// GameClassificationPhysicalGame is for a printable / board / card game
	GameClassificationPhysicalGame GameClassification = "physical_game"
	// GameClassificationSoundtrack is a bunch of music files
	GameClassificationSoundtrack GameClassification = "soundtrack"
	// GameClassificationOther is anything that creators think don't fit in any other category
	GameClassificationOther GameClassification = "other"
	// GameClassificationComic is a comic book (pdf, jpg, specific comic formats, etc.)
	GameClassificationComic GameClassification = "comic"
	// GameClassificationBook is a book (pdf, jpg, specific e-book formats, etc.)
	GameClassificationBook GameClassification = "book"
)

// GameEmbedData contains presentation information for embed games
type GameEmbedData struct {
	// Game this embed info is for
	GameID int64 `json:"gameId" hades:"primary_key"`

	// width of the initial viewport, in pixels
	Width int64 `json:"width"`

	// height of the initial viewport, in pixels
	Height int64 `json:"height"`

	// for itch.io website, whether or not a fullscreen button should be shown
	Fullscreen bool `json:"fullscreen"`
}

// Sale describes a discount for a game.
type Sale struct {
	// Site-wide unique identifier generated by itch.io
	ID int64 `json:"id"`

	// Game this sale is for
	GameID int64 `json:"gameId"`

	// Discount rate in percent.
	// Can be negative, see https://itch.io/updates/introducing-reverse-sales
	Rate float64 `json:"rate"`
	// Timestamp the sale started at
	StartDate time.Time `json:"startDate"`
	// Timestamp the sale ends at
	EndDate time.Time `json:"endDate"`
}

// An Upload is a downloadable file. Some are wharf-enabled, which means
// they're actually a "channel" that may contain multiple builds, pushed
// with <https://github.com/itchio/butler>
type Upload struct {
	// Site-wide unique identifier generated by itch.io
	ID int64 `json:"id"`
	// Storage (hosted, external, etc.)
	Storage UploadStorage `json:"storage"`
	// Host (if external storage)
	Host string `json:"host,omitempty"`
	// Original file name (example: `Overland_x64.zip`)
	Filename string `json:"filename"`
	// Human-friendly name set by developer (example: `Overland for Windows 64-bit`)
	DisplayName string `json:"displayName"`
	// Size of upload in bytes. For wharf-enabled uploads, it's the archive size.
	Size int64 `json:"size"`
	// Name of the wharf channel for this upload, if it's a wharf-enabled upload
	ChannelName string `json:"channelName"`
	// Latest build for this upload, if it's a wharf-enabled upload
	Build *Build `json:"build"`
	// ID of the latest build for this upload, if it's a wharf-enabled upload
	BuildID int64 `json:"buildId,omitempty"`

	// Upload type: default, soundtrack, etc.
	Type UploadType `json:"type"`

	// Is this upload a pre-order placeholder?
	Preorder bool `json:"preorder"`

	// Is this upload a free demo?
	Demo bool `json:"demo"`

	// Platforms this upload is compatible with
	Platforms Platforms `json:"platforms" hades:"squash"`

	// Date this upload was created at
	CreatedAt *time.Time `json:"createdAt"`
	// Date this upload was last updated at (order changed, display name set, etc.)
	UpdatedAt *time.Time `json:"updatedAt"`
}

// UploadStorage describes where an upload file is stored.
type UploadStorage string

const (
	// UploadStorageHosted is a classic upload (web) - no versioning
	UploadStorageHosted UploadStorage = "hosted"
	// UploadStorageBuild is a wharf upload (butler)
	UploadStorageBuild UploadStorage = "build"
	// UploadStorageExternal is an external upload - alllllllll bets are off.
	UploadStorageExternal UploadStorage = "external"
)

// UploadType describes what's in an upload - an executable,
// a web game, some music, etc.
type UploadType string

const (
	// UploadTypeDefault is for executables
	UploadTypeDefault UploadType = "default"

	//----------------
	// embed types
	//----------------

	// UploadTypeFlash is for .swf files
	UploadTypeFlash UploadType = "flash"
	// UploadTypeUnity is for .unity3d files
	UploadTypeUnity UploadType = "unity"
	// UploadTypeJava is for .jar files
	UploadTypeJava UploadType = "java"
	// UploadTypeHTML is for .html files
	UploadTypeHTML UploadType = "html"

	//----------------
	// asorted types
	//----------------

	// UploadTypeSoundtrack is for archives with .mp3/.ogg/.flac/etc files
	UploadTypeSoundtrack UploadType = "soundtrack"
	// UploadTypeBook is for books (epubs, pdfs, etc.)
	UploadTypeBook UploadType = "book"
	// UploadTypeVideo is for videos
	UploadTypeVideo UploadType = "video"
	// UploadTypeDocumentation is for documentation (pdf, maybe uhh doxygen?)
	UploadTypeDocumentation UploadType = "documentation"
	// UploadTypeMod is a bunch of loose files with no clear instructions how to apply them to a game
	UploadTypeMod UploadType = "mod"
	// UploadTypeAudioAssets is a bunch of .ogg/.wav files
	UploadTypeAudioAssets UploadType = "audio_assets"
	// UploadTypeGraphicalAssets is a bunch of .png/.svg/.gif files, maybe some .objs thrown in there
	UploadTypeGraphicalAssets UploadType = "graphical_assets"
	// UploadTypeSourcecode is for source code. No further comments.
	UploadTypeSourcecode UploadType = "sourcecode"
	// UploadTypeOther is for literally anything that isn't an existing category,
	// or for stuff that isn't tagged properly.
	UploadTypeOther UploadType = "other"
)

// A Collection is a set of games, curated by humans.
type Collection struct {
	// Site-wide unique identifier generated by itch.io
	ID int64 `json:"id"`

	// Human-friendly title for collection, for example `Couch coop games`
	Title string `json:"title"`

	// Date this collection was created at
	CreatedAt *time.Time `json:"createdAt"`
	// Date this collection was last updated at (item added, title set, etc.)
	UpdatedAt *time.Time `json:"updatedAt"`

	// Number of games in the collection. This might not be accurate
	// as some games might not be accessible to whoever is asking (project
	// page deleted, visibility level changed, etc.)
	GamesCount int64 `json:"gamesCount"`

	// Games in this collection, with additional info
	CollectionGames []*CollectionGame `json:"collectionGames,omitempty"`

	UserID int64 `json:"userId"`
	User   *User `json:"user,omitempty"`
}

// CollectionGame represents a game's membership for a collection.
type CollectionGame struct {
	CollectionID int64       `json:"collectionId" hades:"primary_key"`
	Collection   *Collection `json:"collection,omitempty"`

	GameID int64 `json:"gameId" hades:"primary_key"`
	Game   *Game `json:"game,omitempty"`

	Position int64 `json:"position"`

	CreatedAt *time.Time `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt"`

	Blurb  string `json:"blurb"`
	UserID int64  `json:"userId"`
}

// A DownloadKey is often generated when a purchase is made, it
// allows downloading uploads for a game that are not available
// for free. It can also be generated by other means.
type DownloadKey struct {
	// Site-wide unique identifier generated by itch.io
	ID int64 `json:"id"`

	// Identifier of the game to which this download key grants access
	GameID int64 `json:"gameId"`

	// Game to which this download key grants access
	Game *Game `json:"game,omitempty"`

	// Date this key was created at (often coincides with purchase time)
	CreatedAt *time.Time `json:"createdAt"`
	// Date this key was last updated at
	UpdatedAt *time.Time `json:"updatedAt"`

	// Identifier of the itch.io user to which this key belongs
	OwnerID int64 `json:"ownerId"`
}

// Build contains information about a specific build
type Build struct {
	// Site-wide unique identifier generated by itch.io
	ID int64 `json:"id"`
	// Identifier of the build before this one on the same channel,
	// or 0 if this is the initial build.
	ParentBuildID int64 `json:"parentBuildId"`
	// State of the build: started, processing, etc.
	State BuildState `json:"state"`

	// Automatically-incremented version number, starting with 1
	Version int64 `json:"version"`
	// Value specified by developer with `--userversion` when pushing a build
	// Might not be unique across builds of a given channel.
	UserVersion string `json:"userVersion"`

	// Files associated with this build - often at least an archive,
	// a signature, and a patch. Some might be missing while the build
	// is still processing or if processing has failed.
	Files []*BuildFile `json:"files"`

	// User who pushed the build
	User *User `json:"user"`
	// Timestamp the build was created at
	CreatedAt *time.Time `json:"createdAt"`
	// Timestamp the build was last updated at
	UpdatedAt *time.Time `json:"updatedAt"`
}

// BuildState describes the state of a build, relative to its initial upload, and
// its processing.
type BuildState string

const (
	// BuildStateStarted is the state of a build from its creation until the initial upload is complete
	BuildStateStarted BuildState = "started"
	// BuildStateProcessing is the state of a build from the initial upload's completion to its fully-processed state.
	// This state does not mean the build is actually being processed right now, it's just queued for processing.
	BuildStateProcessing BuildState = "processing"
	// BuildStateCompleted means the build was successfully processed. Its patch hasn't necessarily been
	// rediff'd yet, but we have the holy (patch,signature,archive) trinity.
	BuildStateCompleted BuildState = "completed"
	// BuildStateFailed means something went wrong with the build. A failing build will not update the channel
	// head and can be requeued by the itch.io team, although if a new build is pushed before they do,
	// that new build will "win".
	BuildStateFailed BuildState = "failed"
)

// BuildFile contains information about a build's "file", which could be its
// archive, its signature, its patch, etc.
type BuildFile struct {
	// Site-wide unique identifier generated by itch.io
	ID int64 `json:"id"`
	// Size of this build file
	Size int64 `json:"size"`
	// State of this file: created, uploading, uploaded, etc.
	State BuildFileState `json:"state"`
	// Type of this build file: archive, signature, patch, etc.
	Type BuildFileType `json:"type"`
	// Subtype of this build file, usually indicates compression
	SubType BuildFileSubType `json:"subType"`

	// Date this build file was created at
	CreatedAt *time.Time `json:"createdAt"`
	// Date this build file was last updated at
	UpdatedAt *time.Time `json:"updatedAt"`
}

// BuildFileState describes the state of a specific file for a build
type BuildFileState string

const (
	// BuildFileStateCreated means the file entry exists on itch.io
	BuildFileStateCreated BuildFileState = "created"
	// BuildFileStateUploading means the file is currently being uploaded to storage
	BuildFileStateUploading BuildFileState = "uploading"
	// BuildFileStateUploaded means the file is ready
	BuildFileStateUploaded BuildFileState = "uploaded"
	// BuildFileStateFailed means the file failed uploading
	BuildFileStateFailed BuildFileState = "failed"
)

// BuildFileType describes the type of a build file: patch, archive, signature, etc.
type BuildFileType string

const (
	// BuildFileTypePatch describes wharf patch files (.pwr)
	BuildFileTypePatch BuildFileType = "patch"
	// BuildFileTypeArchive describes canonical archive form (.zip)
	BuildFileTypeArchive BuildFileType = "archive"
	// BuildFileTypeSignature describes wharf signature files (.pws)
	BuildFileTypeSignature BuildFileType = "signature"
	// BuildFileTypeManifest is reserved
	BuildFileTypeManifest BuildFileType = "manifest"
	// BuildFileTypeUnpacked describes the single file that is in the build (if it was just a single file)
	BuildFileTypeUnpacked BuildFileType = "unpacked"
)

// BuildFileSubType describes the subtype of a build file: mostly its compression
// level. For example, rediff'd patches are "optimized", whereas initial patches are "default"
type BuildFileSubType string

const (
	// BuildFileSubTypeDefault describes default compression (rsync patches)
	BuildFileSubTypeDefault BuildFileSubType = "default"
	// BuildFileSubTypeGzip is reserved
	BuildFileSubTypeGzip BuildFileSubType = "gzip"
	// BuildFileSubTypeOptimized describes optimized compression (rediff'd / bsdiff patches)
	BuildFileSubTypeOptimized BuildFileSubType = "optimized"
)
