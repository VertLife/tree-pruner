package main

// Development constants
// const (
// 	MinSampleSize = 100
// 	MaxTreeSize   = 10000

// 	TreeFilePath   = "sample_trees/%s/single_trees/%s"
// 	OutputBaseDir  = "sample_trees/pruned_treesets/%s/"
// 	OutputBasePath = "sample_trees/pruned_treesets/%s/pruned/%s"
// 	ConfigPath     = "sample_trees/pruned_treesets/%s/config.yaml"
// 	DefaultBucket  = "mol-temp"

// 	DefaultQueue = ""
// )

// Production contants
// Global constants
const (
	MinSampleSize = 100
	MaxTreeSize   = 10000

	TreeFilePath   = "processing/%s/sources/single_trees/%s"
	OutputBaseDir  = "pruned_treesets/%s/"
	OutputBasePath = "pruned_treesets/%s/pruned/%s"
	ConfigPath     = "pruned_treesets/%s/config.yaml"

	DefaultBucket = "data.vertlife.org"
	DefaultQueue  = "mol-tasker-slow-1"
)

// TreeCodes is a map of treeset codes and their names for display
var TreeCodes = map[string]string{
	"EricsonStage2Full":                  "Ericson All Species",
	"EricsonStage1Full":                  "Ericson Sequenced Species",
	"HackettStage2Full":                  "Hackett All Species",
	"HackettStage1Full":                  "Hackett Sequenced Species",
	"Stage2_DecisiveParrot":              "Stage2 Parrot",
	"Stage2_FPTrees_EricsonDecisive":     "Stage2 FP Trees Ericson",
	"Stage2_FPTrees_HackettDecisive":     "Stage2 FP Trees Hackett",
	"Stage2_MayrAll_Ericson_decisive":    "Stage2 MayrAll Ericson",
	"Stage2_MayrParSho_Ericson_decisive": "Stage2 MayrParSho Ericson",
	"Stage2_MayrAll_Hackett_decisive":    "Stage2 MayrAll Hackett",
	"Stage2_MayrParSho_Hackett_decisive": "Stage2 MayrParSho Hackett",
	"Chond_10Cal_full_trees":             "Full resolved 10 fossil  (Set of 10K trees)",
	"Chond_1Cal_full_trees":              "Full resolved 1 fossil  (Set of 10K trees)",
	"Chond_10Cal_sequence_data":          "Sequenced set 10 fossil  (Set of 500 trees)",
	"Chond_1Cal_sequence_data":           "Sequenced set 1 fossil  (Set of 500 trees)",
	"amph_shl_new_Posterior_7238":        "Amphibians Posterior All Species",
	"squam_shl_new_Posterior_9755":       "Squamates Posterior All Species",
}

// TreeSiteCodes show available base trees
var TreeSiteCodes = map[string]string{
	"birdtree":      "BirdTree",
	"sharktree":     "SharkTree",
	"amphibiantree": "AmphibianTree",
	"squamatetree":  "SquamateTree",
}

// TreeSiteUrls show available websites
// Should match TreeSiteCodes
var TreeSiteUrls = map[string]string{
	"birdtree":      "birdtree.org",
	"sharktree":     "sharktree.org",
	"amphibiantree": "vertlife.org",
	"squamatetree":  "vertlife.org",
}

// TreeCitationLong for long citation
var TreeCitationLong = map[string]string{
	"birdtree":      "The global diversity of birds in space and time; W. Jetz, G. H. Thomas, J. B. Joy, K. Hartmann, A. O. Mooers, doi:10.1038/nature11631",
	"sharktree":     "Global priorities for conserving the evolutionary history of sharks, rays, and chimaeras; R.W. Stein, C.G. Mull, T.S. Kuhn, N.C. Aschliman, L.N.K. Davidson, J. B. Joy, G.J. Smith, N.K. Dulvy, A.O. Mooers, doi:10.1038/s41559-017-0448-4",
	"amphibiantree": "Diversification, evolutionary isolation, and imperilment across the amphibian tree of life; Jetz & Pyron, doi:10.1038/s41559-018-0515-5",
	"squamatetree":  "Fully-sampled phylogenies of squamates reveal evolutionary patterns in threat status; Tonini JFR, Beard KH, Ferreira RB, Jetz W, Pyron RA, doi:10.1016/j.biocon.2016.03.039",
}

// TreeCitationShort for short citation
var TreeCitationShort = map[string]string{
	"birdtree":      "Jetz et al. 2012",
	"sharktree":     "Stein et al. 2018",
	"amphibiantree": "Jetz & Pyron 2018",
	"squamatetree":  "Tonini et al. 2016",
}

// JobResult represents the JSON output
type JobResult struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}
