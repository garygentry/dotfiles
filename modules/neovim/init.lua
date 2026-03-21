-- init.lua - Neovim Configuration
-- Managed by dotfiles

-- =============================================================================
-- Bootstrap lazy.nvim plugin manager
-- =============================================================================

local lazypath = vim.fn.stdpath("data") .. "/lazy/lazy.nvim"
if not (vim.uv or vim.loop).fs_stat(lazypath) then
  vim.fn.system({
    "git",
    "clone",
    "--filter=blob:none",
    "https://github.com/folke/lazy.nvim.git",
    "--branch=stable",
    lazypath,
  })
end
vim.opt.rtp:prepend(lazypath)

-- =============================================================================
-- Basic Options
-- =============================================================================

-- Line numbers
vim.opt.number = true
vim.opt.relativenumber = true

-- Tabs and indentation
vim.opt.tabstop = 2
vim.opt.shiftwidth = 2
vim.opt.softtabstop = 2
vim.opt.expandtab = true
vim.opt.smartindent = true
vim.opt.autoindent = true

-- Search
vim.opt.hlsearch = true
vim.opt.incsearch = true
vim.opt.ignorecase = true
vim.opt.smartcase = true

-- Clipboard (use system clipboard)
vim.opt.clipboard = "unnamedplus"

-- Appearance
vim.opt.termguicolors = true
vim.opt.signcolumn = "yes"
vim.opt.cursorline = true
vim.opt.scrolloff = 8
vim.opt.sidescrolloff = 8
vim.opt.wrap = false
vim.opt.showmode = false       -- redundant with lualine
vim.opt.pumheight = 10         -- limit completion menu height
vim.opt.fileencoding = "utf-8"
vim.opt.conceallevel = 0       -- show markdown syntax as-is

-- Splits
vim.opt.splitbelow = true
vim.opt.splitright = true

-- File handling
vim.opt.swapfile = false
vim.opt.backup = false
vim.opt.undofile = true
vim.opt.undodir = vim.fn.stdpath("data") .. "/undo"

-- Performance
vim.opt.updatetime = 250
vim.opt.timeoutlen = 300

-- Completion
vim.opt.completeopt = { "menu", "menuone", "noselect" }

-- =============================================================================
-- Leader Key
-- =============================================================================

vim.g.mapleader = " "
vim.g.maplocalleader = " "

-- Disable netrw (using nvim-tree instead)
vim.g.loaded_netrw = 1
vim.g.loaded_netrwPlugin = 1

-- =============================================================================
-- Plugin Setup (via lazy.nvim)
-- =============================================================================

require("lazy").setup({

  -- ---------------------------------------------------------------------------
  -- Colorscheme
  -- ---------------------------------------------------------------------------
  {
    "catppuccin/nvim",
    name = "catppuccin",
    priority = 1000,
    config = function()
      require("catppuccin").setup({
        flavour = "mocha",
        integrations = {
          cmp = true,
          gitsigns = true,
          nvimtree = true,
          telescope = { enabled = true },
          treesitter = true,
          which_key = true,
          mason = true,
          lsp_trouble = true,
          indent_blankline = { enabled = true },
        },
      })
      vim.cmd.colorscheme("catppuccin")
    end,
  },

  -- ---------------------------------------------------------------------------
  -- File explorer
  -- ---------------------------------------------------------------------------
  {
    "nvim-tree/nvim-tree.lua",
    dependencies = { "nvim-tree/nvim-web-devicons" },
    config = function()
      require("nvim-tree").setup({
        view = { width = 35 },
        renderer = {
          group_empty = true,
          icons = {
            show = { file = true, folder = true, folder_arrow = true, git = true },
          },
        },
        filters = { dotfiles = false },
        git = { enable = true, ignore = false },
      })
    end,
  },

  -- ---------------------------------------------------------------------------
  -- Fuzzy finder
  -- ---------------------------------------------------------------------------
  {
    "nvim-telescope/telescope.nvim",
    dependencies = {
      "nvim-lua/plenary.nvim",
      { "nvim-telescope/telescope-fzf-native.nvim", build = "make" },
    },
    config = function()
      local telescope = require("telescope")
      local actions = require("telescope.actions")

      telescope.setup({
        defaults = {
          mappings = {
            i = {
              ["<C-k>"] = actions.move_selection_previous,
              ["<C-j>"] = actions.move_selection_next,
              ["<C-q>"] = actions.send_selected_to_qflist + actions.open_qflist,
            },
          },
          file_ignore_patterns = { "node_modules", ".git/", "dist/", ".turbo/" },
        },
      })

      telescope.load_extension("fzf")
    end,
  },

  -- ---------------------------------------------------------------------------
  -- Treesitter - syntax highlighting & indentation
  -- ---------------------------------------------------------------------------
  {
    "nvim-treesitter/nvim-treesitter",
    build = ":TSUpdate",
    config = function()
      require("nvim-treesitter").setup({
        ensure_installed = {
          "lua", "vim", "vimdoc",
          "go", "gomod",
          "typescript", "tsx", "javascript",
          "json", "jsonc", "yaml", "toml",
          "html", "css",
          "markdown", "markdown_inline",
          "bash", "sql",
        },
        auto_install = true,
      })
    end,
  },

  -- ---------------------------------------------------------------------------
  -- Status line
  -- ---------------------------------------------------------------------------
  {
    "nvim-lualine/lualine.nvim",
    dependencies = { "nvim-tree/nvim-web-devicons", "catppuccin" },
    config = function()
      require("lualine").setup({
        options = {
          theme = "catppuccin-nvim",
          component_separators = { left = "", right = "" },
          section_separators = { left = "", right = "" },
        },
        sections = {
          lualine_a = { "mode" },
          lualine_b = { "branch", "diff", "diagnostics" },
          lualine_c = { { "filename", path = 1 } },
          lualine_x = { "encoding", "fileformat", "filetype" },
          lualine_y = { "progress" },
          lualine_z = { "location" },
        },
      })
    end,
  },

  -- ---------------------------------------------------------------------------
  -- Git
  -- ---------------------------------------------------------------------------
  {
    "lewis6991/gitsigns.nvim",
    config = function()
      require("gitsigns").setup({
        signs = {
          add          = { text = "▎" },
          change       = { text = "▎" },
          delete       = { text = "" },
          topdelete    = { text = "" },
          changedelete = { text = "▎" },
          untracked    = { text = "▎" },
        },
        on_attach = function(bufnr)
          local gs = package.loaded.gitsigns
          local function map(mode, l, r, opts)
            opts = opts or {}
            opts.buffer = bufnr
            vim.keymap.set(mode, l, r, opts)
          end

          -- Navigation
          map("n", "]c", function()
            if vim.wo.diff then return "]c" end
            vim.schedule(function() gs.next_hunk() end)
            return "<Ignore>"
          end, { expr = true })

          map("n", "[c", function()
            if vim.wo.diff then return "[c" end
            vim.schedule(function() gs.prev_hunk() end)
            return "<Ignore>"
          end, { expr = true })

          -- Actions
          map("n", "<leader>gs", gs.stage_hunk)
          map("n", "<leader>gr", gs.reset_hunk)
          map("n", "<leader>gp", gs.preview_hunk)
          map("n", "<leader>gb", function() gs.blame_line({ full = true }) end)
          map("n", "<leader>gd", gs.diffthis)
        end,
      })
    end,
  },

  -- ---------------------------------------------------------------------------
  -- LSP
  -- ---------------------------------------------------------------------------
  {
    "williamboman/mason.nvim",
    config = function()
      require("mason").setup({
        ui = { border = "rounded" },
      })

      -- Auto-install LSP servers and tools via mason-registry
      local ensure_installed = {
        "typescript-language-server",
        "lua-language-server",
        "gopls",
        "html-lsp",
        "css-lsp",
        "json-lsp",
        "yaml-language-server",
        "eslint-lsp",
      }
      local registry = require("mason-registry")
      registry.refresh(function()
        for _, name in ipairs(ensure_installed) do
          local ok, pkg = pcall(registry.get_package, name)
          if ok and not pkg:is_installed() then
            pkg:install()
          end
        end
      end)
    end,
  },

  -- ---------------------------------------------------------------------------
  -- Completion
  -- ---------------------------------------------------------------------------
  {
    "hrsh7th/nvim-cmp",
    dependencies = {
      "hrsh7th/cmp-nvim-lsp",
      "hrsh7th/cmp-buffer",
      "hrsh7th/cmp-path",
      "hrsh7th/cmp-cmdline",
      "L3MON4D3/LuaSnip",
      "saadparwaiz1/cmp_luasnip",
      "rafamadriz/friendly-snippets",  -- pre-built snippet collection
    },
    config = function()
      local cmp = require("cmp")
      local luasnip = require("luasnip")

      -- Load VSCode-style snippets from friendly-snippets
      require("luasnip.loaders.from_vscode").lazy_load()

      cmp.setup({
        snippet = {
          expand = function(args)
            luasnip.lsp_expand(args.body)
          end,
        },
        window = {
          completion = cmp.config.window.bordered(),
          documentation = cmp.config.window.bordered(),
        },
        mapping = cmp.mapping.preset.insert({
          ["<C-k>"]     = cmp.mapping.select_prev_item(),
          ["<C-j>"]     = cmp.mapping.select_next_item(),
          ["<C-b>"]     = cmp.mapping.scroll_docs(-4),
          ["<C-f>"]     = cmp.mapping.scroll_docs(4),
          ["<C-Space>"] = cmp.mapping.complete(),
          ["<C-e>"]     = cmp.mapping.abort(),
          ["<CR>"]      = cmp.mapping.confirm({ select = false }),
          ["<Tab>"] = cmp.mapping(function(fallback)
            if cmp.visible() then
              cmp.select_next_item()
            elseif luasnip.expand_or_jumpable() then
              luasnip.expand_or_jump()
            else
              fallback()
            end
          end, { "i", "s" }),
          ["<S-Tab>"] = cmp.mapping(function(fallback)
            if cmp.visible() then
              cmp.select_prev_item()
            elseif luasnip.jumpable(-1) then
              luasnip.jump(-1)
            else
              fallback()
            end
          end, { "i", "s" }),
        }),
        sources = cmp.config.sources({
          { name = "nvim_lsp", priority = 1000 },
          { name = "luasnip",  priority = 750 },
          { name = "buffer",   priority = 500 },
          { name = "path",     priority = 250 },
        }),
        formatting = {
          format = function(entry, item)
            local source_labels = {
              nvim_lsp = "[LSP]",
              luasnip  = "[Snip]",
              buffer   = "[Buf]",
              path     = "[Path]",
            }
            item.menu = source_labels[entry.source.name] or ""
            return item
          end,
        },
      })

      -- Command-line completion
      cmp.setup.cmdline({ "/", "?" }, {
        mapping = cmp.mapping.preset.cmdline(),
        sources = { { name = "buffer" } },
      })

      cmp.setup.cmdline(":", {
        mapping = cmp.mapping.preset.cmdline(),
        sources = cmp.config.sources(
          { { name = "path" } },
          { { name = "cmdline" } }
        ),
      })
    end,
  },

  -- ---------------------------------------------------------------------------
  -- Formatting
  -- ---------------------------------------------------------------------------
  {
    "stevearc/conform.nvim",
    config = function()
      require("conform").setup({
        formatters_by_ft = {
          lua        = { "stylua" },
          javascript = { "prettierd", "prettier", stop_after_first = true },
          typescript = { "prettierd", "prettier", stop_after_first = true },
          javascriptreact = { "prettierd", "prettier", stop_after_first = true },
          typescriptreact = { "prettierd", "prettier", stop_after_first = true },
          json       = { "prettierd", "prettier", stop_after_first = true },
          yaml       = { "prettierd", "prettier", stop_after_first = true },
          markdown   = { "prettierd", "prettier", stop_after_first = true },
          html       = { "prettierd", "prettier", stop_after_first = true },
          css        = { "prettierd", "prettier", stop_after_first = true },
        },
        format_on_save = {
          timeout_ms = 500,
          lsp_fallback = true,
        },
      })
    end,
  },

  -- ---------------------------------------------------------------------------
  -- Linting
  -- ---------------------------------------------------------------------------
  {
    "mfussenegger/nvim-lint",
    config = function()
      require("lint").linters_by_ft = {
        javascript      = { "eslint_d" },
        typescript      = { "eslint_d" },
        javascriptreact = { "eslint_d" },
        typescriptreact = { "eslint_d" },
      }

      vim.api.nvim_create_autocmd({ "BufWritePost", "BufReadPost", "InsertLeave" }, {
        callback = function()
          require("lint").try_lint()
        end,
      })
    end,
  },

  -- ---------------------------------------------------------------------------
  -- Which-key (keybinding hints)
  -- ---------------------------------------------------------------------------
  {
    "folke/which-key.nvim",
    event = "VeryLazy",
    config = function()
      require("which-key").setup({
        win = { border = "rounded" },
      })

      -- Register group labels
      require("which-key").add({
        { "<leader>g", group = "git" },
        { "<leader>f", group = "find/telescope" },
        { "<leader>l", group = "lsp" },
        { "<leader>t", group = "trouble" },
      })
    end,
  },

  -- ---------------------------------------------------------------------------
  -- Trouble (diagnostics panel)
  -- ---------------------------------------------------------------------------
  {
    "folke/trouble.nvim",
    dependencies = { "nvim-tree/nvim-web-devicons" },
    config = function()
      require("trouble").setup({})
    end,
  },

  -- ---------------------------------------------------------------------------
  -- Autopairs
  -- ---------------------------------------------------------------------------
  {
    "windwp/nvim-autopairs",
    event = "InsertEnter",
    config = function()
      local autopairs = require("nvim-autopairs")
      autopairs.setup({ check_ts = true })

      -- Integrate with cmp
      local cmp_autopairs = require("nvim-autopairs.completion.cmp")
      local cmp = require("cmp")
      cmp.event:on("confirm_done", cmp_autopairs.on_confirm_done())
    end,
  },

  -- ---------------------------------------------------------------------------
  -- Commenting
  -- ---------------------------------------------------------------------------
  {
    "numToStr/Comment.nvim",
    config = function()
      require("Comment").setup()
    end,
  },

  -- ---------------------------------------------------------------------------
  -- Tmux/Neovim navigation
  -- ---------------------------------------------------------------------------
  {
    "christoomey/vim-tmux-navigator",
    cmd = { "TmuxNavigateLeft", "TmuxNavigateDown", "TmuxNavigateUp", "TmuxNavigateRight" },
    keys = {
      { "<C-h>", "<cmd>TmuxNavigateLeft<cr>" },
      { "<C-j>", "<cmd>TmuxNavigateDown<cr>" },
      { "<C-k>", "<cmd>TmuxNavigateUp<cr>" },
      { "<C-l>", "<cmd>TmuxNavigateRight<cr>" },
    },
  },

  -- ---------------------------------------------------------------------------
  -- Indent guides
  -- ---------------------------------------------------------------------------
  {
    "lukas-reineke/indent-blankline.nvim",
    main = "ibl",
    config = function()
      require("ibl").setup({
        indent = { char = "│" },
        scope = { enabled = true },
      })
    end,
  },

})

-- =============================================================================
-- LSP Configuration (native vim.lsp API, no lspconfig plugin needed)
-- =============================================================================

-- LSP keybindings via LspAttach autocmd
vim.api.nvim_create_autocmd("LspAttach", {
  callback = function(ev)
    local map = function(keys, func, desc)
      vim.keymap.set("n", keys, func, { buffer = ev.buf, desc = "LSP: " .. desc })
    end

    map("gd",         vim.lsp.buf.definition,       "Go to Definition")
    map("gD",         vim.lsp.buf.declaration,      "Go to Declaration")
    map("gr",         vim.lsp.buf.references,       "Go to References")
    map("gi",         vim.lsp.buf.implementation,   "Go to Implementation")
    map("gt",         vim.lsp.buf.type_definition,  "Go to Type Definition")
    map("K",          vim.lsp.buf.hover,            "Hover Documentation")
    map("<C-k>",      vim.lsp.buf.signature_help,   "Signature Help")
    map("<leader>rn", vim.lsp.buf.rename,           "Rename Symbol")
    map("<leader>ca", vim.lsp.buf.code_action,      "Code Action")
    map("<leader>fs", require("telescope.builtin").lsp_document_symbols, "Document Symbols")

    -- ESLint: fix on save
    local client = vim.lsp.get_client_by_id(ev.data.client_id)
    if client and client.name == "eslint" then
      vim.api.nvim_create_autocmd("BufWritePre", {
        buffer = ev.buf,
        command = "EslintFixAll",
      })
    end
  end,
})

-- Global capabilities for all servers
local capabilities = require("cmp_nvim_lsp").default_capabilities()
vim.lsp.config("*", {
  capabilities = capabilities,
})

-- TypeScript
vim.lsp.config("ts_ls", {
  settings = {
    typescript = {
      inlayHints = {
        includeInlayParameterNameHints = "all",
        includeInlayFunctionParameterTypeHints = true,
        includeInlayVariableTypeHints = true,
        includeInlayPropertyDeclarationTypeHints = true,
        includeInlayFunctionLikeReturnTypeHints = true,
      },
    },
    javascript = {
      inlayHints = {
        includeInlayParameterNameHints = "all",
        includeInlayFunctionParameterTypeHints = true,
      },
    },
  },
})

-- Lua (with Neovim globals awareness)
vim.lsp.config("lua_ls", {
  settings = {
    Lua = {
      diagnostics = { globals = { "vim" } },
      workspace = {
        library = vim.api.nvim_get_runtime_file("", true),
        checkThirdParty = false,
      },
      telemetry = { enable = false },
    },
  },
})

-- Go
vim.lsp.config("gopls", {
  settings = {
    gopls = {
      analyses = { unusedparams = true },
      staticcheck = true,
    },
  },
})

-- Enable all configured servers
vim.lsp.enable({
  "ts_ls", "lua_ls", "eslint", "gopls",
  "html", "cssls", "jsonls", "yamlls",
})

-- Diagnostic display config
vim.diagnostic.config({
  virtual_text = { prefix = "●" },
  signs = true,
  underline = true,
  update_in_insert = false,
  severity_sort = true,
  float = { border = "rounded", source = true },
})

-- =============================================================================
-- Keybindings
-- =============================================================================

local keymap = vim.keymap.set
local opts = { noremap = true, silent = true }

-- Clear search highlighting
keymap("n", "<leader>h", ":nohlsearch<CR>", opts)

-- Stay centered when navigating search results
keymap("n", "n", "nzzzv", opts)
keymap("n", "N", "Nzzzv", opts)

-- Stay centered on C-d/C-u
keymap("n", "<C-d>", "<C-d>zz", opts)
keymap("n", "<C-u>", "<C-u>zz", opts)

-- Window resize
keymap("n", "<C-Up>",    ":resize -2<CR>",          opts)
keymap("n", "<C-Down>",  ":resize +2<CR>",          opts)
keymap("n", "<C-Left>",  ":vertical resize -2<CR>", opts)
keymap("n", "<C-Right>", ":vertical resize +2<CR>", opts)

-- Buffer navigation
keymap("n", "<S-l>", ":bnext<CR>",     opts)
keymap("n", "<S-h>", ":bprevious<CR>", opts)
keymap("n", "<leader>x", ":bdelete<CR>", opts)

-- File explorer
keymap("n", "<leader>e", ":NvimTreeToggle<CR>", opts)

-- Telescope
keymap("n", "<leader>ff", ":Telescope find_files<CR>",              opts)
keymap("n", "<leader>fg", ":Telescope live_grep<CR>",               opts)
keymap("n", "<leader>fb", ":Telescope buffers<CR>",                 opts)
keymap("n", "<leader>fr", ":Telescope oldfiles<CR>",                opts)
keymap("n", "<leader>fd", ":Telescope diagnostics<CR>",             opts)
keymap("n", "<leader>fk", ":Telescope keymaps<CR>",                 opts)

-- Visual mode: move lines
keymap("v", "J", ":m '>+1<CR>gv=gv", opts)
keymap("v", "K", ":m '<-2<CR>gv=gv", opts)

-- Visual mode: paste without clobbering register
keymap("v", "p", '"_dP', opts)

-- Visual mode: stay in indent mode
keymap("v", "<", "<gv", opts)
keymap("v", ">", ">gv", opts)

-- Diagnostics navigation
keymap("n", "[d", function()
  (vim.diagnostic.jump or vim.diagnostic.goto_prev)({ count = -1 })
end, opts)
keymap("n", "]d", function()
  (vim.diagnostic.jump or vim.diagnostic.goto_next)({ count = 1 })
end, opts)
keymap("n", "<leader>ld", vim.diagnostic.open_float, opts)

-- Trouble
keymap("n", "<leader>tt", ":Trouble diagnostics toggle<CR>", opts)
keymap("n", "<leader>td", ":Trouble diagnostics toggle filter.buf=0<CR>", opts)

-- Format buffer manually
keymap("n", "<leader>lf", function()
  require("conform").format({ async = true, lsp_fallback = true })
end, { noremap = true, silent = true, desc = "Format buffer" })

-- Quick save
keymap("n", "<C-s>", ":w<CR>", opts)
keymap("i", "<C-s>", "<Esc>:w<CR>a", opts)