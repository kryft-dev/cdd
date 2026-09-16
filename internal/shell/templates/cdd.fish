# cdd init fish | source
function cdd --description 'Jump to a recent Project'
    switch "$argv[1]"
        case {{range .Passthrough}}{{.}} {{end}}--help -h --version -V
            command cdd $argv
        case '*'
            set -l result (command cdd pick $argv)
            set -l cdd_status $status
            if test $cdd_status -ne 0
                return $cdd_status
            end
            test -n "$result"
            and cd -- $result
    end
end
