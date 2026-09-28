$untracked = git ls-files --others --exclude-standard
$modified = git diff --name-only

$allFiles = $untracked + $modified

$count = 0
foreach ($file in $allFiles) {
    if (![string]::IsNullOrWhiteSpace($file) -and (Test-Path $file)) {
        git add $file
        if ($file -eq "README.md") {
            git commit -m "update: enhance README documentation for clarity"
        } elseif ($file -eq ".gitignore") {
            git commit -m "update: update gitignore with appropriate exclusions"
        } elseif ($file -like "backend/*") {
            $basename = Split-Path $file -Leaf
            git commit -m "feat: add backend implementation for $basename"
        } elseif ($file -like "frontend/*") {
            $basename = Split-Path $file -Leaf
            git commit -m "feat: scaffold frontend functionality for $basename"
        } else {
            $basename = Split-Path $file -Leaf
            git commit -m "setup: integrate project asset $basename"
        }
        $count++
    }
}
Write-Output "Total commits created: $count"
