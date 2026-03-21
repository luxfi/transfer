package router

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/luxfi/transfer/pkg/comms"
	"github.com/luxfi/transfer/pkg/disclosures"
	"github.com/luxfi/transfer/pkg/dividends"
	"github.com/luxfi/transfer/pkg/filings"
	"github.com/luxfi/transfer/pkg/ledger"
	"github.com/luxfi/transfer/pkg/restrictions"
	"github.com/luxfi/transfer/pkg/shareholder"
	"github.com/luxfi/transfer/pkg/store"
	"github.com/luxfi/transfer/pkg/voting"
)

// New creates a chi router with all transfer agent endpoints mounted.
func New(
	shareholders *shareholder.Service,
	ledgerSvc *ledger.Service,
	restrictionsSvc *restrictions.Service,
	disclosuresSvc *disclosures.Service,
	commsSvc *comms.Service,
	dividendsSvc *dividends.Service,
	filingsSvc *filings.Service,
	votingSvc *voting.Service,
) chi.Router {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// --- Shareholders ---
	r.Route("/shareholders", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, req *http.Request) {
			filter := store.ShareholderFilter{
				Type:  req.URL.Query().Get("type"),
				Query: req.URL.Query().Get("q"),
			}
			if v := req.URL.Query().Get("accredited"); v != "" {
				b := v == "true"
				filter.Accredited = &b
			}
			list, err := shareholders.List(req.Context(), filter)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, list)
		})

		r.Post("/", func(w http.ResponseWriter, req *http.Request) {
			var sh store.Shareholder
			if err := json.NewDecoder(req.Body).Decode(&sh); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			if err := shareholders.Create(req.Context(), &sh); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusCreated, sh)
		})

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				sh, err := shareholders.Get(req.Context(), chi.URLParam(req, "id"))
				if err != nil {
					writeErr(w, http.StatusNotFound, err)
					return
				}
				writeJSON(w, http.StatusOK, sh)
			})

			r.Patch("/", func(w http.ResponseWriter, req *http.Request) {
				var sh store.Shareholder
				if err := json.NewDecoder(req.Body).Decode(&sh); err != nil {
					writeErr(w, http.StatusBadRequest, err)
					return
				}
				sh.ID = chi.URLParam(req, "id")
				if err := shareholders.Update(req.Context(), &sh); err != nil {
					writeErr(w, http.StatusBadRequest, err)
					return
				}
				writeJSON(w, http.StatusOK, sh)
			})

			r.Get("/holdings", func(w http.ResponseWriter, req *http.Request) {
				holdings, err := shareholders.GetHoldings(req.Context(), chi.URLParam(req, "id"))
				if err != nil {
					writeErr(w, http.StatusNotFound, err)
					return
				}
				writeJSON(w, http.StatusOK, holdings)
			})

			r.Get("/restrictions", func(w http.ResponseWriter, req *http.Request) {
				list, err := restrictionsSvc.List(req.Context(), chi.URLParam(req, "id"))
				if err != nil {
					writeErr(w, http.StatusInternalServerError, err)
					return
				}
				writeJSON(w, http.StatusOK, list)
			})
		})
	})

	// --- Transfers / Ledger ---
	r.Route("/transfers", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, req *http.Request) {
			filter := store.TransferFilter{
				ShareholderID: req.URL.Query().Get("shareholder_id"),
				ShareClassID:  req.URL.Query().Get("share_class_id"),
				Type:          req.URL.Query().Get("type"),
			}
			list, err := ledgerSvc.ListTransfers(req.Context(), filter)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, list)
		})

		r.Post("/", func(w http.ResponseWriter, req *http.Request) {
			var t store.Transfer
			if err := json.NewDecoder(req.Body).Decode(&t); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			if err := ledgerSvc.RecordTransfer(req.Context(), &t); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusCreated, t)
		})

		r.Get("/{id}", func(w http.ResponseWriter, req *http.Request) {
			// Filter by ID — returns first match
			list, err := ledgerSvc.ListTransfers(req.Context(), store.TransferFilter{})
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
			id := chi.URLParam(req, "id")
			for _, t := range list {
				if t.ID == id {
					writeJSON(w, http.StatusOK, t)
					return
				}
			}
			writeErr(w, http.StatusNotFound, nil)
		})
	})

	r.Get("/ledger/{shareClassId}", func(w http.ResponseWriter, req *http.Request) {
		list, err := ledgerSvc.ListTransfers(req.Context(), store.TransferFilter{
			ShareClassID: chi.URLParam(req, "shareClassId"),
		})
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	})

	// --- Disclosures ---
	r.Route("/disclosures", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, req *http.Request) {
			filter := store.DisclosureFilter{
				Type:          req.URL.Query().Get("type"),
				ShareholderID: req.URL.Query().Get("shareholder_id"),
			}
			list, err := disclosuresSvc.List(req.Context(), filter)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, list)
		})

		r.Post("/", func(w http.ResponseWriter, req *http.Request) {
			var d store.Disclosure
			if err := json.NewDecoder(req.Body).Decode(&d); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			if err := disclosuresSvc.Create(req.Context(), &d); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusCreated, d)
		})

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				d, err := disclosuresSvc.Get(req.Context(), chi.URLParam(req, "id"))
				if err != nil {
					writeErr(w, http.StatusNotFound, err)
					return
				}
				writeJSON(w, http.StatusOK, d)
			})

			r.Post("/deliver", func(w http.ResponseWriter, req *http.Request) {
				var body struct {
					ShareholderID string `json:"shareholder_id"`
				}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					writeErr(w, http.StatusBadRequest, err)
					return
				}
				if err := disclosuresSvc.Deliver(req.Context(), chi.URLParam(req, "id"), body.ShareholderID); err != nil {
					writeErr(w, http.StatusBadRequest, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "delivered"})
			})

			r.Post("/acknowledge", func(w http.ResponseWriter, req *http.Request) {
				var body struct {
					ShareholderID string `json:"shareholder_id"`
				}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					writeErr(w, http.StatusBadRequest, err)
					return
				}
				if err := disclosuresSvc.Acknowledge(req.Context(), chi.URLParam(req, "id"), body.ShareholderID); err != nil {
					writeErr(w, http.StatusBadRequest, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
			})
		})
	})

	// --- Notices ---
	r.Route("/notices", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, req *http.Request) {
			filter := store.NoticeFilter{
				Type: req.URL.Query().Get("type"),
			}
			list, err := commsSvc.List(req.Context(), filter)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, list)
		})

		r.Post("/", func(w http.ResponseWriter, req *http.Request) {
			var n store.Notice
			if err := json.NewDecoder(req.Body).Decode(&n); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			if err := commsSvc.Create(req.Context(), &n); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusCreated, n)
		})

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				n, err := commsSvc.Get(req.Context(), chi.URLParam(req, "id"))
				if err != nil {
					writeErr(w, http.StatusNotFound, err)
					return
				}
				writeJSON(w, http.StatusOK, n)
			})

			r.Post("/send", func(w http.ResponseWriter, req *http.Request) {
				if err := commsSvc.Send(req.Context(), chi.URLParam(req, "id")); err != nil {
					writeErr(w, http.StatusBadRequest, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
			})
		})
	})

	// --- Dividends ---
	r.Route("/dividends", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, req *http.Request) {
			filter := store.DividendFilter{
				ShareClassID: req.URL.Query().Get("share_class_id"),
				Status:       req.URL.Query().Get("status"),
			}
			list, err := dividendsSvc.List(req.Context(), filter)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, list)
		})

		r.Post("/", func(w http.ResponseWriter, req *http.Request) {
			var d store.Dividend
			if err := json.NewDecoder(req.Body).Decode(&d); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			if err := dividendsSvc.Create(req.Context(), &d); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusCreated, d)
		})

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				d, err := dividendsSvc.Get(req.Context(), chi.URLParam(req, "id"))
				if err != nil {
					writeErr(w, http.StatusNotFound, err)
					return
				}
				writeJSON(w, http.StatusOK, d)
			})

			r.Post("/calculate", func(w http.ResponseWriter, req *http.Request) {
				d, err := dividendsSvc.Calculate(req.Context(), chi.URLParam(req, "id"))
				if err != nil {
					writeErr(w, http.StatusBadRequest, err)
					return
				}
				writeJSON(w, http.StatusOK, d)
			})

			r.Post("/pay", func(w http.ResponseWriter, req *http.Request) {
				d, err := dividendsSvc.Pay(req.Context(), chi.URLParam(req, "id"))
				if err != nil {
					writeErr(w, http.StatusBadRequest, err)
					return
				}
				writeJSON(w, http.StatusOK, d)
			})
		})
	})

	// --- Filings ---
	r.Route("/filings", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, req *http.Request) {
			filter := store.FilingFilter{
				Type:         req.URL.Query().Get("type"),
				Jurisdiction: req.URL.Query().Get("jurisdiction"),
				Status:       req.URL.Query().Get("status"),
			}
			list, err := filingsSvc.List(req.Context(), filter)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, list)
		})

		r.Post("/", func(w http.ResponseWriter, req *http.Request) {
			var f store.Filing
			if err := json.NewDecoder(req.Body).Decode(&f); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			if err := filingsSvc.Create(req.Context(), &f); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusCreated, f)
		})

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				f, err := filingsSvc.Get(req.Context(), chi.URLParam(req, "id"))
				if err != nil {
					writeErr(w, http.StatusNotFound, err)
					return
				}
				writeJSON(w, http.StatusOK, f)
			})

			r.Patch("/", func(w http.ResponseWriter, req *http.Request) {
				var f store.Filing
				if err := json.NewDecoder(req.Body).Decode(&f); err != nil {
					writeErr(w, http.StatusBadRequest, err)
					return
				}
				f.ID = chi.URLParam(req, "id")
				if err := filingsSvc.Update(req.Context(), &f); err != nil {
					writeErr(w, http.StatusBadRequest, err)
					return
				}
				writeJSON(w, http.StatusOK, f)
			})
		})
	})

	// --- Voting ---
	r.Route("/proposals", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, req *http.Request) {
			filter := store.ProposalFilter{
				Status: req.URL.Query().Get("status"),
			}
			list, err := votingSvc.ListProposals(req.Context(), filter)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, list)
		})

		r.Post("/", func(w http.ResponseWriter, req *http.Request) {
			var p store.Proposal
			if err := json.NewDecoder(req.Body).Decode(&p); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			if err := votingSvc.CreateProposal(req.Context(), &p); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusCreated, p)
		})

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				p, err := votingSvc.GetProposal(req.Context(), chi.URLParam(req, "id"))
				if err != nil {
					writeErr(w, http.StatusNotFound, err)
					return
				}
				writeJSON(w, http.StatusOK, p)
			})

			r.Post("/vote", func(w http.ResponseWriter, req *http.Request) {
				var v store.Vote
				if err := json.NewDecoder(req.Body).Decode(&v); err != nil {
					writeErr(w, http.StatusBadRequest, err)
					return
				}
				v.ProposalID = chi.URLParam(req, "id")
				if err := votingSvc.CastVote(req.Context(), &v); err != nil {
					writeErr(w, http.StatusBadRequest, err)
					return
				}
				writeJSON(w, http.StatusCreated, v)
			})

			r.Get("/results", func(w http.ResponseWriter, req *http.Request) {
				results, err := votingSvc.GetResults(req.Context(), chi.URLParam(req, "id"))
				if err != nil {
					writeErr(w, http.StatusNotFound, err)
					return
				}
				writeJSON(w, http.StatusOK, results)
			})
		})
	})

	// --- Restrictions ---
	r.Route("/restrictions", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, req *http.Request) {
			list, err := restrictionsSvc.List(req.Context(), req.URL.Query().Get("shareholder_id"))
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, list)
		})

		r.Post("/", func(w http.ResponseWriter, req *http.Request) {
			var rst store.Restriction
			if err := json.NewDecoder(req.Body).Decode(&rst); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			if err := restrictionsSvc.Create(req.Context(), &rst); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusCreated, rst)
		})

		r.Delete("/{id}", func(w http.ResponseWriter, req *http.Request) {
			if err := restrictionsSvc.Delete(req.Context(), chi.URLParam(req, "id")); err != nil {
				writeErr(w, http.StatusNotFound, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

		r.Post("/check", func(w http.ResponseWriter, req *http.Request) {
			var body struct {
				FromShareholderID string `json:"from_shareholder_id"`
				ToShareholderID   string `json:"to_shareholder_id"`
				ShareClassID      string `json:"share_class_id"`
				Quantity          int64  `json:"quantity"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				writeErr(w, http.StatusBadRequest, err)
				return
			}
			check, err := restrictionsSvc.Check(req.Context(), body.FromShareholderID, body.ToShareholderID, body.ShareClassID, body.Quantity)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, check)
		})
	})

	return r
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	msg := http.StatusText(status)
	if err != nil {
		msg = err.Error()
	}
	writeJSON(w, status, map[string]string{"error": msg})
}

