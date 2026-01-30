package openrtb

import (
	"encoding/json"
)

// OpenRTB 2.1 core objects.
// Spec: OpenRTB API Specification Version 2.1 (IAB Tech Lab).

// BidRequest represents the top-level bid request object.
type BidRequest struct {
	ID          string          `json:"id"`
	Imp         []Imp           `json:"imp"`
	Site        *Site           `json:"site,omitempty"`
	App         *App            `json:"app,omitempty"`
	Device      *Device         `json:"device,omitempty"`
	User        *User           `json:"user,omitempty"`
	Test        int             `json:"test,omitempty"`
	AuctionType int             `json:"at,omitempty"`
	TMax        int             `json:"tmax,omitempty"`
	WSeat       []string        `json:"wseat,omitempty"`
	BSeat       []string        `json:"bseat,omitempty"`
	AllIMPS     int             `json:"allimps,omitempty"`
	Cur         []string        `json:"cur,omitempty"`
	WLang       []string        `json:"wlang,omitempty"`
	BCategory   []string        `json:"bcat,omitempty"`
	BAdv        []string        `json:"badv,omitempty"`
	Regs        *Regs           `json:"regs,omitempty"`
	Ext         json.RawMessage `json:"ext,omitempty"`
}

// Imp represents an impression being offered for auction.
type Imp struct {
	ID                string          `json:"id"`
	Metric            []Metric        `json:"metric,omitempty"`
	Banner            *Banner         `json:"banner,omitempty"`
	Video             *Video          `json:"video,omitempty"`
	Audio             *Audio          `json:"audio,omitempty"`
	Native            json.RawMessage `json:"native,omitempty"` // Native 1.0 request as JSON object.
	PMP               *PMP            `json:"pmp,omitempty"`
	DisplayManager    string          `json:"displaymanager,omitempty"`
	DisplayManagerVer string          `json:"displaymanagerver,omitempty"`
	Instl             int             `json:"instl,omitempty"`
	TagID             string          `json:"tagid,omitempty"`
	BidFloor          float64         `json:"bidfloor,omitempty"`
	BidFloorCur       string          `json:"bidfloorcur,omitempty"`
	Secure            int             `json:"secure,omitempty"`
	IFRAMEBuster      []string        `json:"iframebuster,omitempty"`
	Ext               json.RawMessage `json:"ext,omitempty"`
}

// Metric represents a set of metrics for an impression.
type Metric struct {
	Type   string  `json:"type"`
	Value  float64 `json:"value"`
	Vendor string  `json:"vendor,omitempty"`
	Ext    any     `json:"ext,omitempty"`
}

// Banner represents banner-type impression details.
type Banner struct {
	W              int             `json:"w,omitempty"`
	H              int             `json:"h,omitempty"`
	WMax           int             `json:"wmax,omitempty"`
	HMax           int             `json:"hmax,omitempty"`
	WMin           int             `json:"wmin,omitempty"`
	HMin           int             `json:"hmin,omitempty"`
	ID             string          `json:"id,omitempty"`
	Pos            int             `json:"pos,omitempty"`
	BType          []int           `json:"btype,omitempty"`
	BAttr          []int           `json:"battr,omitempty"`
	MIME           []string        `json:"mimes,omitempty"`
	TopFrame       int             `json:"topframe,omitempty"`
	ExpDir         []int           `json:"expdir,omitempty"`
	API            []int           `json:"api,omitempty"`
	Ext            json.RawMessage `json:"ext,omitempty"`
	Format         []Format        `json:"format,omitempty"` // Introduced in later OpenRTB; harmless if unused.
	BlockedAttr    []int           `json:"blockedattr,omitempty"`
	BlockedCat     []string        `json:"blockedcat,omitempty"`
	BlockedAdv     []string        `json:"blockedadv,omitempty"`
	BlockedCreative []string       `json:"blockedcreative,omitempty"`
}

// Format represents a banner size supported by the impression.
type Format struct {
	W   int `json:"w,omitempty"`
	H   int `json:"h,omitempty"`
	WRT int `json:"wratio,omitempty"`
	HRT int `json:"hratio,omitempty"`
	Ext any `json:"ext,omitempty"`
}

// Video represents video-type impression details.
type Video struct {
	MIME            []string        `json:"mimes,omitempty"`
	MinDuration     int             `json:"minduration,omitempty"`
	MaxDuration     int             `json:"maxduration,omitempty"`
	Protocols       []int           `json:"protocols,omitempty"`
	Protocol        int             `json:"protocol,omitempty"`
	W               int             `json:"w,omitempty"`
	H               int             `json:"h,omitempty"`
	StartDelay      int             `json:"startdelay,omitempty"`
	Linearity       int             `json:"linearity,omitempty"`
	Sequence        int             `json:"sequence,omitempty"`
	BAttr           []int           `json:"battr,omitempty"`
	MaxExtended     int             `json:"maxextended,omitempty"`
	MinBitrate      int             `json:"minbitrate,omitempty"`
	MaxBitrate      int             `json:"maxbitrate,omitempty"`
	BoxingAllowed   int             `json:"boxingallowed,omitempty"`
	PlaybackMethod  []int           `json:"playbackmethod,omitempty"`
	Delivery        []int           `json:"delivery,omitempty"`
	Pos             int             `json:"pos,omitempty"`
	CompanionAd     []Banner        `json:"companionad,omitempty"`
	API             []int           `json:"api,omitempty"`
	CompanionType   []int           `json:"companiontype,omitempty"`
	Ext             json.RawMessage `json:"ext,omitempty"`
}

// Audio represents audio-type impression details.
type Audio struct {
	MIME           []string        `json:"mimes,omitempty"`
	MinDuration    int             `json:"minduration,omitempty"`
	MaxDuration    int             `json:"maxduration,omitempty"`
	Protocols      []int           `json:"protocols,omitempty"`
	StartDelay     int             `json:"startdelay,omitempty"`
	Sequence       int             `json:"sequence,omitempty"`
	BAttr          []int           `json:"battr,omitempty"`
	MaxExtended    int             `json:"maxextended,omitempty"`
	MinBitrate     int             `json:"minbitrate,omitempty"`
	MaxBitrate     int             `json:"maxbitrate,omitempty"`
	Delivery       []int           `json:"delivery,omitempty"`
	CompanionAd    []Banner        `json:"companionad,omitempty"`
	API            []int           `json:"api,omitempty"`
	CompanionType  []int           `json:"companiontype,omitempty"`
	Ext            json.RawMessage `json:"ext,omitempty"`
}

// PMP represents private marketplace options for an impression.
type PMP struct {
	PrivateAuction int             `json:"private_auction,omitempty"`
	Deals          []Deal          `json:"deals,omitempty"`
	Ext            json.RawMessage `json:"ext,omitempty"`
}

// Deal represents a PMP deal.
type Deal struct {
	ID          string          `json:"id"`
	BidFloor    float64         `json:"bidfloor,omitempty"`
	BidFloorCur string          `json:"bidfloorcur,omitempty"`
	WSeat       []string        `json:"wseat,omitempty"`
	WAdv        []string        `json:"wadomain,omitempty"`
	AT          int             `json:"at,omitempty"`
	Ext         json.RawMessage `json:"ext,omitempty"`
}

// Site represents website details for impressions.
type Site struct {
	ID            string          `json:"id,omitempty"`
	Name          string          `json:"name,omitempty"`
	Domain        string          `json:"domain,omitempty"`
	Cat           []string        `json:"cat,omitempty"`
	SectionCat    []string        `json:"sectioncat,omitempty"`
	PageCat       []string        `json:"pagecat,omitempty"`
	Page          string          `json:"page,omitempty"`
	Ref           string          `json:"ref,omitempty"`
	Search        string          `json:"search,omitempty"`
	Mobile        int             `json:"mobile,omitempty"`
	PrivacyPolicy int             `json:"privacypolicy,omitempty"`
	Publisher     *Publisher      `json:"publisher,omitempty"`
	Content       *Content        `json:"content,omitempty"`
	Keywords      string          `json:"keywords,omitempty"`
	Ext           json.RawMessage `json:"ext,omitempty"`
}

// App represents app details for impressions.
type App struct {
	ID            string          `json:"id,omitempty"`
	Name          string          `json:"name,omitempty"`
	Domain        string          `json:"domain,omitempty"`
	Cat           []string        `json:"cat,omitempty"`
	SectionCat    []string        `json:"sectioncat,omitempty"`
	PageCat       []string        `json:"pagecat,omitempty"`
	Version       string          `json:"ver,omitempty"`
	Bundle        string          `json:"bundle,omitempty"`
	StoreURL      string          `json:"storeurl,omitempty"`
	Publisher     *Publisher      `json:"publisher,omitempty"`
	Content       *Content        `json:"content,omitempty"`
	Keywords      string          `json:"keywords,omitempty"`
	PrivacyPolicy int             `json:"privacypolicy,omitempty"`
	Paid          int             `json:"paid,omitempty"`
	Ext           json.RawMessage `json:"ext,omitempty"`
}

// Publisher represents publisher details.
type Publisher struct {
	ID     string          `json:"id,omitempty"`
	Name   string          `json:"name,omitempty"`
	Cat    []string        `json:"cat,omitempty"`
	Domain string          `json:"domain,omitempty"`
	Ext    json.RawMessage `json:"ext,omitempty"`
}

// Content represents content details.
type Content struct {
	ID           string          `json:"id,omitempty"`
	Episode      int             `json:"episode,omitempty"`
	Title        string          `json:"title,omitempty"`
	Series       string          `json:"series,omitempty"`
	Season       string          `json:"season,omitempty"`
	Producer     *Producer       `json:"producer,omitempty"`
	URL          string          `json:"url,omitempty"`
	Cat          []string        `json:"cat,omitempty"`
	ProdQ        int             `json:"prodq,omitempty"`
	VideoQuality int             `json:"videoquality,omitempty"`
	Context      int             `json:"context,omitempty"`
	ContentRating string         `json:"contentrating,omitempty"`
	UserRating   string          `json:"userrating,omitempty"`
	QAGMediaRating int           `json:"qagmediarating,omitempty"`
	Keywords     string          `json:"keywords,omitempty"`
	LiveStream   int             `json:"livestream,omitempty"`
	SourceRelationship int       `json:"sourcerelationship,omitempty"`
	Length       int             `json:"len,omitempty"`
	Language     string          `json:"language,omitempty"`
	Embeddable   int             `json:"embeddable,omitempty"`
	Ext          json.RawMessage `json:"ext,omitempty"`
}

// Producer represents content producer details.
type Producer struct {
	ID     string          `json:"id,omitempty"`
	Name   string          `json:"name,omitempty"`
	Cat    []string        `json:"cat,omitempty"`
	Domain string          `json:"domain,omitempty"`
	Ext    json.RawMessage `json:"ext,omitempty"`
}

// Device represents device details.
type Device struct {
	UA            string          `json:"ua,omitempty"`
	Geo           *Geo            `json:"geo,omitempty"`
	DNT           int             `json:"dnt,omitempty"`
	LMT           int             `json:"lmt,omitempty"`
	IP            string          `json:"ip,omitempty"`
	IPv6          string          `json:"ipv6,omitempty"`
	DeviceType    int             `json:"devicetype,omitempty"`
	Make          string          `json:"make,omitempty"`
	Model         string          `json:"model,omitempty"`
	OS            string          `json:"os,omitempty"`
	OSV           string          `json:"osv,omitempty"`
	HWVersion     string          `json:"hwv,omitempty"`
	H             int             `json:"h,omitempty"`
	W             int             `json:"w,omitempty"`
	PPI           int             `json:"ppi,omitempty"`
	PXRatio       float64         `json:"pxratio,omitempty"`
	JS            int             `json:"js,omitempty"`
	FlashVer      string          `json:"flashver,omitempty"`
	Language      string          `json:"language,omitempty"`
	Carrier       string          `json:"carrier,omitempty"`
	ConnectionType int            `json:"connectiontype,omitempty"`
	IFA           string          `json:"ifa,omitempty"`
	DIDSHA1       string          `json:"didsha1,omitempty"`
	DIDMD5        string          `json:"didmd5,omitempty"`
	DPIDSHA1      string          `json:"dpidsha1,omitempty"`
	DPIDMD5       string          `json:"dpidmd5,omitempty"`
	MACSHA1       string          `json:"macsha1,omitempty"`
	MACMD5        string          `json:"macmd5,omitempty"`
	Ext           json.RawMessage `json:"ext,omitempty"`
}

// Geo represents geographic location of the device/user.
type Geo struct {
	Lat           float64         `json:"lat,omitempty"`
	Lon           float64         `json:"lon,omitempty"`
	Type          int             `json:"type,omitempty"`
	Accuracy      int             `json:"accuracy,omitempty"`
	LastFix       int             `json:"lastfix,omitempty"`
	IPService     int             `json:"ipservice,omitempty"`
	Country       string          `json:"country,omitempty"`
	Region        string          `json:"region,omitempty"`
	RegionFIPS104 string          `json:"regionfips104,omitempty"`
	Metro         string          `json:"metro,omitempty"`
	City          string          `json:"city,omitempty"`
	Zip           string          `json:"zip,omitempty"`
	UTCOffset     int             `json:"utcoffset,omitempty"`
	Ext           json.RawMessage `json:"ext,omitempty"`
}

// User represents user information.
type User struct {
	ID         string          `json:"id,omitempty"`
	BuyerUID   string          `json:"buyeruid,omitempty"`
	YOB        int             `json:"yob,omitempty"`
	Gender     string          `json:"gender,omitempty"`
	Keywords   string          `json:"keywords,omitempty"`
	CustomData string          `json:"customdata,omitempty"`
	Geo        *Geo            `json:"geo,omitempty"`
	Data       []Data          `json:"data,omitempty"`
	Ext        json.RawMessage `json:"ext,omitempty"`
}

// Data represents data about the user from a data provider.
type Data struct {
	ID      string          `json:"id,omitempty"`
	Name    string          `json:"name,omitempty"`
	Segment []Segment       `json:"segment,omitempty"`
	Ext     json.RawMessage `json:"ext,omitempty"`
}

// Segment represents a data segment from a data provider.
type Segment struct {
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Value string          `json:"value,omitempty"`
	Ext   json.RawMessage `json:"ext,omitempty"`
}

// Regs represents regulations (e.g. COPPA).
type Regs struct {
	COPPA int             `json:"coppa,omitempty"`
	Ext   json.RawMessage `json:"ext,omitempty"`
}

// -----------------------
// Bid response (OpenRTB)
// -----------------------

// BidResponse represents the top-level bid response object.
type BidResponse struct {
	ID      string          `json:"id"`
	SeatBid []SeatBid       `json:"seatbid,omitempty"`
	BidID   string          `json:"bidid,omitempty"`
	Cur     string          `json:"cur,omitempty"`
	CustomData string       `json:"customdata,omitempty"`
	NBR     int             `json:"nbr,omitempty"`
	Ext     json.RawMessage `json:"ext,omitempty"`
}

// SeatBid groups bids by bidder seat.
type SeatBid struct {
	Bid   []Bid           `json:"bid,omitempty"`
	Seat  string          `json:"seat,omitempty"`
	Group int             `json:"group,omitempty"`
	Ext   json.RawMessage `json:"ext,omitempty"`
}

// Bid represents a bid for an impression.
type Bid struct {
	ID         string          `json:"id"`
	ImpID      string          `json:"impid"`
	Price      float64         `json:"price"`
	AdID       string          `json:"adid,omitempty"`
	NURL       string          `json:"nurl,omitempty"`
	Adm        string          `json:"adm,omitempty"`
	Adomain    []string        `json:"adomain,omitempty"`
	Bundle     string          `json:"bundle,omitempty"`
	IURL       string          `json:"iurl,omitempty"`
	CID        string          `json:"cid,omitempty"`
	CRID       string          `json:"crid,omitempty"`
	Cat        []string        `json:"cat,omitempty"`
	Attr       []int           `json:"attr,omitempty"`
	API        int             `json:"api,omitempty"`
	Protocol   int             `json:"protocol,omitempty"`
	QAGMediaRating int         `json:"qagmediarating,omitempty"`
	DealID     string          `json:"dealid,omitempty"`
	H          int             `json:"h,omitempty"`
	W          int             `json:"w,omitempty"`
	Ext        json.RawMessage `json:"ext,omitempty"`
}
