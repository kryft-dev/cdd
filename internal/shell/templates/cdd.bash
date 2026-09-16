# eval "$(cdd init bash)"
cdd() {
    case "${1-}" in
        {{range .Passthrough}}{{.}}|{{end}}--help|-h|--version|-V) \command cdd "$@" ;;
        *)
            local result
            result="$(\command cdd pick "$@")"
            local cdd_status=$?
            if [ "$cdd_status" -ne 0 ]; then
                return "$cdd_status"
            fi
            [ -n "$result" ] && \builtin cd -- "$result"
            ;;
    esac
}
