[CmdletBinding()]
param(
  [int]$Port = 18086,
  [string]$BaseDir = ''
)

$ErrorActionPreference = 'Stop'

function Get-UatCookieHeader {
  param(
    [Microsoft.PowerShell.Commands.WebRequestSession]$Session,
    [string]$BaseUrl
  )
  $cookies = $Session.Cookies.GetCookies($BaseUrl)
  ($cookies | ForEach-Object { "$($_.Name)=$($_.Value)" }) -join '; '
}

function Get-UatCookieValue {
  param(
    [Microsoft.PowerShell.Commands.WebRequestSession]$Session,
    [string]$BaseUrl,
    [string]$Name
  )
  ($Session.Cookies.GetCookies($BaseUrl) | Where-Object { $_.Name -eq $Name } | Select-Object -First 1).Value
}

function Write-UatTextFile {
  param(
    [string]$Path,
    [string]$Content
  )
  $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
  [System.IO.File]::WriteAllText($Path, $Content, $utf8NoBom)
}

function Invoke-UatJson {
  param(
    [Microsoft.PowerShell.Commands.WebRequestSession]$Session,
    [string]$BaseUrl,
    [string]$Path,
    [ValidateSet('GET', 'POST', 'PUT', 'PATCH', 'DELETE')]
    [string]$Method = 'GET',
    [object]$Body,
    [string]$CsrfToken
  )

  $headers = @{}
  if ($CsrfToken) {
    $headers['X-CSRF-Token'] = $CsrfToken
  }

  $requestParams = @{
    UseBasicParsing = $true
    WebSession      = $Session
    Uri             = "$BaseUrl$Path"
    Method          = $Method
    Headers         = $headers
    ErrorAction     = 'Stop'
  }
  if ($PSBoundParameters.ContainsKey('Body')) {
    $requestParams['ContentType'] = 'application/json'
    if ($null -ne $Body) {
      $requestParams['Body'] = ($Body | ConvertTo-Json -Depth 20 -Compress)
    }
  }
  $response = Invoke-WebRequest @requestParams
  if ([string]::IsNullOrWhiteSpace($response.Content)) {
    return $null
  }
  return $response.Content | ConvertFrom-Json
}

function Invoke-UatMultipartUpload {
  param(
    [string]$BaseUrl,
    [Microsoft.PowerShell.Commands.WebRequestSession]$Session,
    [string]$CsrfToken,
    [string]$SubmissionId,
    [string]$FilePath,
    [string]$ArtifactKind,
    [string]$ContentType
  )

  $cookieHeader = Get-UatCookieHeader -Session $Session -BaseUrl $BaseUrl
  $url = "$BaseUrl/api/v1/student/submissions/$SubmissionId/artifacts"
  $args = @(
    '--silent',
    '--show-error',
    '--fail',
    '-X', 'POST',
    '--cookie', $cookieHeader,
    '-H', "X-CSRF-Token: $CsrfToken",
    '-F', "artifact_kind=$ArtifactKind",
    '-F', "file=@$FilePath;type=$ContentType",
    $url
  )
  $output = & curl.exe @args
  if ($LASTEXITCODE -ne 0) {
    throw "multipart upload failed for $FilePath"
  }
  if ([string]::IsNullOrWhiteSpace($output)) {
    return $null
  }
  return $output | ConvertFrom-Json
}

function New-MinimalPdfBytes {
  param(
    [string]$Text = 'UAT PDF'
  )
  $escape = $Text.Replace('\', '\\').Replace('(', '\(').Replace(')', '\)')
  $content = "BT /F1 24 Tf 50 150 Td ($escape) Tj ET`n"
  $encoding = [System.Text.Encoding]::ASCII
  $stream = New-Object System.IO.MemoryStream
  $writer = New-Object System.IO.StreamWriter($stream, $encoding, 1024, $true)
  $objects = @(
    "1 0 obj`n<< /Type /Catalog /Pages 2 0 R >>`nendobj`n",
    "2 0 obj`n<< /Type /Pages /Kids [3 0 R] /Count 1 >>`nendobj`n",
    "3 0 obj`n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>`nendobj`n",
    "4 0 obj`n<< /Length $([System.Text.Encoding]::ASCII.GetByteCount($content)) >>`nstream`n$content`nendstream`nendobj`n",
    "5 0 obj`n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>`nendobj`n"
  )
  $offsets = @()
  $header = "%PDF-1.4`n"
  $writer.Write($header)
  $writer.Flush()
  foreach ($obj in $objects) {
    $offsets += $stream.Length
    $writer.Write($obj)
    $writer.Flush()
  }
  $xrefStart = $stream.Length
  $xref = New-Object System.Text.StringBuilder
  [void]$xref.AppendLine('xref')
  [void]$xref.AppendLine('0 6')
  [void]$xref.AppendLine('0000000000 65535 f ')
  foreach ($offset in $offsets) {
    [void]$xref.AppendLine(('{0:0000000000} 00000 n ' -f $offset))
  }
  [void]$xref.AppendLine('trailer')
  [void]$xref.AppendLine('<< /Size 6 /Root 1 0 R >>')
  [void]$xref.AppendLine('startxref')
  [void]$xref.AppendLine($xrefStart)
  [void]$xref.AppendLine('%%EOF')
  $writer.Write($xref.ToString())
  $writer.Flush()
  return $stream.ToArray()
}

function New-UatSampleFiles {
  param(
    [string]$Root
  )

  $sampleDir = Join-Path $Root 'samples'
  New-Item -ItemType Directory -Force -Path $sampleDir | Out-Null

  $files = [ordered]@{}

  $report = Join-Path $sampleDir 'report.md'
  Write-UatTextFile -Path $report -Content @"
# UAT Report

This is a sample report.
"@
  $files.report = $report

  $docxDir = Join-Path $sampleDir 'docx'
  New-Item -ItemType Directory -Force -Path (Join-Path $docxDir '_rels'), (Join-Path $docxDir 'word') | Out-Null
  Write-UatTextFile -Path (Join-Path $docxDir '[Content_Types].xml') -Content @'
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>
'@
  Write-UatTextFile -Path (Join-Path $docxDir '_rels/.rels') -Content @'
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>
'@
  Write-UatTextFile -Path (Join-Path $docxDir 'word/document.xml') -Content @'
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>UAT DOCX</w:t></w:r></w:p>
    <w:sectPr><w:pgSz w:w="12240" w:h="15840"/></w:sectPr>
  </w:body>
</w:document>
'@
  $docx = Join-Path $sampleDir 'report.docx'
  if (Test-Path $docx) { Remove-Item $docx -Force }
  Add-Type -AssemblyName System.IO.Compression
  Add-Type -AssemblyName System.IO.Compression.FileSystem
  $docxStream = [System.IO.File]::Open($docx, [System.IO.FileMode]::Create)
  try {
    $docxArchive = New-Object System.IO.Compression.ZipArchive($docxStream, [System.IO.Compression.ZipArchiveMode]::Create, $false)
    try {
      [System.IO.Compression.ZipFileExtensions]::CreateEntryFromFile($docxArchive, (Join-Path $docxDir '[Content_Types].xml'), '[Content_Types].xml') | Out-Null
      [System.IO.Compression.ZipFileExtensions]::CreateEntryFromFile($docxArchive, (Join-Path $docxDir '_rels/.rels'), '_rels/.rels') | Out-Null
      [System.IO.Compression.ZipFileExtensions]::CreateEntryFromFile($docxArchive, (Join-Path $docxDir 'word/document.xml'), 'word/document.xml') | Out-Null
    } finally {
      $docxArchive.Dispose()
    }
  } finally {
    $docxStream.Dispose()
  }
  $files.docx = $docx

  $pdf = Join-Path $sampleDir 'report.pdf'
  [System.IO.File]::WriteAllBytes($pdf, (New-MinimalPdfBytes -Text 'UAT PDF'))
  $files.pdf = $pdf

  $png = Join-Path $sampleDir 'shot.png'
  [System.IO.File]::WriteAllBytes($png, [Convert]::FromBase64String('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/nV0AAAAASUVORK5CYII='))
  $files.png = $png

  $zipRoot = Join-Path $sampleDir 'code'
  New-Item -ItemType Directory -Force -Path $zipRoot | Out-Null
  Write-UatTextFile -Path (Join-Path $zipRoot 'main.go') -Content @"
package main
func main() {}
"@
  Write-UatTextFile -Path (Join-Path $zipRoot 'README.md') -Content @"
# Sample archive
"@
  $zip = Join-Path $sampleDir 'code.zip'
  if (Test-Path $zip) { Remove-Item $zip -Force }
  Compress-Archive -Path (Join-Path $zipRoot '*') -DestinationPath $zip -Force
  $files.zip = $zip

  return $files
}

function Wait-For-Health {
  param(
    [string]$BaseUrl,
    [int]$TimeoutSeconds = 30
  )
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  do {
    try {
      $response = Invoke-WebRequest -UseBasicParsing -Uri "$BaseUrl/health/live" -TimeoutSec 2
      if ($response.StatusCode -eq 200) {
        return
      }
    } catch {
      Start-Sleep -Milliseconds 500
    }
  } while ((Get-Date) -lt $deadline)
  throw "server did not become healthy at $BaseUrl"
}

$repoRoot = if ([string]::IsNullOrWhiteSpace($BaseDir)) { (Get-Location).Path } else { $BaseDir }
$uatRoot = Join-Path $repoRoot ('tmp\uat-' + [Guid]::NewGuid().ToString('n'))
New-Item -ItemType Directory -Force -Path $uatRoot, (Join-Path $uatRoot 'storage') | Out-Null
$dbPath = Join-Path $uatRoot 'app.db'
$runtimePath = Join-Path $uatRoot 'runtime.json'
$logPath = Join-Path $uatRoot 'server.log'
$baseUrl = "http://127.0.0.1:$Port"

if (-not (Test-Path (Join-Path $repoRoot 'tmp-server.exe'))) {
  go build -tags webui -o (Join-Path $repoRoot 'tmp-server.exe') .\cmd\server
}

$job = Start-Job -ScriptBlock {
  param($root, $dbPath, $runtimePath, $storagePath, $port, $logPath)
  Set-Location $root
  $env:DB_DRIVER = 'sqlite'
  $env:SQLITE_PATH = $dbPath
  $env:RUNTIME_CONFIG_PATH = $runtimePath
  $env:STORAGE_ROOT = $storagePath
  $env:AUTO_MIGRATE = 'true'
  $env:HTTP_ADDR = "127.0.0.1:$port"
  & (Join-Path $root 'tmp-server.exe') *> $logPath
} -ArgumentList $repoRoot, $dbPath, $runtimePath, (Join-Path $uatRoot 'storage'), $Port, $logPath

try {
  Wait-For-Health -BaseUrl $baseUrl

  $adminSession = New-Object Microsoft.PowerShell.Commands.WebRequestSession
  $bootstrap = Invoke-UatJson -Session $adminSession -BaseUrl $baseUrl -Path '/api/v1/bootstrap/admin' -Method POST -Body @{
    username = 'admin1'
    display_name = 'Admin One'
    employee_no = 'A001'
    password = 'test-pass'
  }
  $adminCsrf = Get-UatCookieValue -Session $adminSession -BaseUrl $baseUrl -Name 'loong64_b1_csrf'
  $adminMe = Invoke-UatJson -Session $adminSession -BaseUrl $baseUrl -Path '/api/v1/me'

  $headers = @{ }
  if ($adminCsrf) { $headers['X-CSRF-Token'] = $adminCsrf }
  $teacher = Invoke-UatJson -Session $adminSession -BaseUrl $baseUrl -Path '/api/v1/admin/users' -Method POST -CsrfToken $adminCsrf -Body @{
    username = 'teacher1'
    display_name = 'Teacher One'
    employee_no = 'T001'
    password = 'teacher-pass'
    roles = @('teacher')
  }
  $student = Invoke-UatJson -Session $adminSession -BaseUrl $baseUrl -Path '/api/v1/admin/users' -Method POST -CsrfToken $adminCsrf -Body @{
    username = 'student1'
    display_name = 'Student One'
    student_no = 'S001'
    password = 'student-pass'
    roles = @('student')
  }
  $class = Invoke-UatJson -Session $adminSession -BaseUrl $baseUrl -Path '/api/v1/admin/classes' -Method POST -CsrfToken $adminCsrf -Body @{
    code = 'SE2401'
    name = 'Software Engineering 2401'
    grade_year = 2024
    major = 'Software Engineering'
  }
  $course = Invoke-UatJson -Session $adminSession -BaseUrl $baseUrl -Path '/api/v1/admin/courses' -Method POST -CsrfToken $adminCsrf -Body @{
    code = 'SE-LAB-01'
    name = 'Software Lab 1'
    term = '2026-spring'
  }
  Invoke-UatJson -Session $adminSession -BaseUrl $baseUrl -Path "/api/v1/admin/courses/$($course.id)/classes" -Method PUT -CsrfToken $adminCsrf -Body @{ class_id = $class.id } | Out-Null
  Invoke-UatJson -Session $adminSession -BaseUrl $baseUrl -Path "/api/v1/admin/courses/$($course.id)/teachers" -Method PUT -CsrfToken $adminCsrf -Body @{ teacher_id = $teacher.id } | Out-Null
  Invoke-UatJson -Session $adminSession -BaseUrl $baseUrl -Path "/api/v1/admin/courses/$($course.id)/enrollments" -Method PUT -CsrfToken $adminCsrf -Body @{ student_id = $student.id; class_id = $class.id } | Out-Null

  $teacherSession = New-Object Microsoft.PowerShell.Commands.WebRequestSession
  $teacherLogin = Invoke-UatJson -Session $teacherSession -BaseUrl $baseUrl -Path '/api/v1/auth/login' -Method POST -Body @{
    username = 'teacher1'
    password = 'teacher-pass'
  }
  $teacherCsrf = Get-UatCookieValue -Session $teacherSession -BaseUrl $baseUrl -Name 'loong64_b1_csrf'
  $template = Invoke-UatJson -Session $teacherSession -BaseUrl $baseUrl -Path '/api/v1/teacher/rubric-templates' -Method POST -CsrfToken $teacherCsrf -Body @{
    name = 'MVP Template'
    description = 'Base scoring template'
  }
  $version = Invoke-UatJson -Session $teacherSession -BaseUrl $baseUrl -Path "/api/v1/teacher/rubric-templates/$($template.id)/versions" -Method POST -CsrfToken $teacherCsrf -Body @{
    weight_mode = 'strict_100'
    metrics = @(
      @{ code = 'quality'; name = 'Code Quality'; weight_bps = 4000; max_score = 40; sort_order = 1 },
      @{ code = 'docs'; name = 'Docs'; weight_bps = 3000; max_score = 30; sort_order = 2 },
      @{ code = 'feature'; name = 'Feature'; weight_bps = 3000; max_score = 30; sort_order = 3 }
    )
  }
  Invoke-UatJson -Session $teacherSession -BaseUrl $baseUrl -Path "/api/v1/teacher/rubric-template-versions/$($version.version.id)/publish" -Method POST -CsrfToken $teacherCsrf -Body @{} | Out-Null
  $experiment = Invoke-UatJson -Session $teacherSession -BaseUrl $baseUrl -Path "/api/v1/teacher/courses/$($course.id)/experiments" -Method POST -CsrfToken $teacherCsrf -Body @{
    title = 'LoongArch deployment lab'
    description = 'Submit report, screenshot and code archive.'
    rubric_version_id = $version.version.id
    submission_spec = @{ required_artifacts = @('report', 'screenshot', 'code_archive') }
  }
  Invoke-UatJson -Session $teacherSession -BaseUrl $baseUrl -Path "/api/v1/teacher/experiments/$($experiment.id)/publish" -Method POST -CsrfToken $teacherCsrf -Body @{} | Out-Null

  $studentSession = New-Object Microsoft.PowerShell.Commands.WebRequestSession
  Invoke-UatJson -Session $studentSession -BaseUrl $baseUrl -Path '/api/v1/auth/login' -Method POST -Body @{
    username = 'student1'
    password = 'student-pass'
  } | Out-Null
  $studentCsrf = Get-UatCookieValue -Session $studentSession -BaseUrl $baseUrl -Name 'loong64_b1_csrf'
  $studentExperiments = Invoke-UatJson -Session $studentSession -BaseUrl $baseUrl -Path '/api/v1/student/experiments'
  $submission = Invoke-UatJson -Session $studentSession -BaseUrl $baseUrl -Path "/api/v1/student/experiments/$($experiment.id)/submissions" -Method POST -CsrfToken $studentCsrf -Body @{}

  $samples = New-UatSampleFiles -Root $uatRoot
  Invoke-UatMultipartUpload -BaseUrl $baseUrl -Session $studentSession -CsrfToken $studentCsrf -SubmissionId $submission.id -FilePath $samples.report -ArtifactKind 'report' -ContentType 'text/plain' | Out-Null
  Invoke-UatMultipartUpload -BaseUrl $baseUrl -Session $studentSession -CsrfToken $studentCsrf -SubmissionId $submission.id -FilePath $samples.docx -ArtifactKind 'document' -ContentType 'application/octet-stream' | Out-Null
  Invoke-UatMultipartUpload -BaseUrl $baseUrl -Session $studentSession -CsrfToken $studentCsrf -SubmissionId $submission.id -FilePath $samples.pdf -ArtifactKind 'document' -ContentType 'application/pdf' | Out-Null
  Invoke-UatMultipartUpload -BaseUrl $baseUrl -Session $studentSession -CsrfToken $studentCsrf -SubmissionId $submission.id -FilePath $samples.png -ArtifactKind 'screenshot' -ContentType 'image/png' | Out-Null
  Invoke-UatMultipartUpload -BaseUrl $baseUrl -Session $studentSession -CsrfToken $studentCsrf -SubmissionId $submission.id -FilePath $samples.zip -ArtifactKind 'code_archive' -ContentType 'application/zip' | Out-Null

  $submissionDetail = Invoke-UatJson -Session $teacherSession -BaseUrl $baseUrl -Path "/api/v1/teacher/submissions/$($submission.id)"
  $evaluation = Invoke-UatJson -Session $teacherSession -BaseUrl $baseUrl -Path "/api/v1/teacher/submissions/$($submission.id)/evaluations/initial" -Method POST -CsrfToken $teacherCsrf -Body @{ mode = 'rule_and_llm' }
  $reviewPayload = @{
    evaluation_result_id = $evaluation.result.id
    teacher_comment = 'UAT review'
    scores = @(
      @{ metric_code = 'quality'; final_score = 36; source = 'rule'; source_metric_score_id = $evaluation.scores[0].id; adjustment_reason = 'UAT smoke'; comment = 'OK' },
      @{ metric_code = 'docs'; final_score = 28; source = 'rule'; source_metric_score_id = $evaluation.scores[1].id; adjustment_reason = 'UAT smoke'; comment = 'OK' },
      @{ metric_code = 'feature'; final_score = 29; source = 'rule'; source_metric_score_id = $evaluation.scores[2].id; adjustment_reason = 'UAT smoke'; comment = 'OK' }
    )
  }
  $review = Invoke-UatJson -Session $teacherSession -BaseUrl $baseUrl -Path "/api/v1/teacher/submissions/$($submission.id)/review" -Method PUT -CsrfToken $teacherCsrf -Body $reviewPayload
  $published = Invoke-UatJson -Session $teacherSession -BaseUrl $baseUrl -Path "/api/v1/teacher/submissions/$($submission.id)/review/publish" -Method POST -CsrfToken $teacherCsrf -Body @{ confirm = $true }
  $studentReview = Invoke-UatJson -Session $studentSession -BaseUrl $baseUrl -Path "/api/v1/student/submissions/$($submission.id)/review"

  $reportExport = Invoke-UatJson -Session $teacherSession -BaseUrl $baseUrl -Path "/api/v1/teacher/submissions/$($submission.id)/report-exports" -Method POST -CsrfToken $teacherCsrf -Body @{ format = 'html' }
  $experimentExport = Invoke-UatJson -Session $teacherSession -BaseUrl $baseUrl -Path "/api/v1/teacher/experiments/$($experiment.id)/report-exports" -Method POST -CsrfToken $teacherCsrf -Body @{ format = 'csv' }
  $courseExport = Invoke-UatJson -Session $teacherSession -BaseUrl $baseUrl -Path "/api/v1/teacher/courses/$($course.id)/report-exports" -Method POST -CsrfToken $teacherCsrf -Body @{ format = 'pdf' }

  $summary = [pscustomobject]@{
    bootstrap = $bootstrap
    admin_me = $adminMe
    teacher_id = $teacher.id
    student_id = $student.id
    class_id = $class.id
    course_id = $course.id
    template_id = $template.id
    version_id = $version.version.id
    experiment_id = $experiment.id
    submission_id = $submission.id
    evaluation_id = $evaluation.result.id
    review_status = $review.review.status
    published_status = $published.review.status
    student_review_status = $studentReview.review.status
    report_export_id = $reportExport.id
    experiment_export_id = $experimentExport.id
    course_export_id = $courseExport.id
    student_experiments = $studentExperiments.items.Count
  }
  $summary | ConvertTo-Json -Depth 10
}