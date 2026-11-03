SOFF='/usr/lib/libreoffice/program/soffice --headless --norestore -env:UserInstallation=file:///tmp/lo-profile --convert-to'
D=$1
docker run --rm -v "$D:/work" --entrypoint sh gotenberg/gotenberg:snapshot -c "mkdir -p /work/out; cd /work;
for f in odt doc rtf docx html pdf; do $SOFF \$f --outdir /work/out /work/memo.fodt; done;
for f in ods xls xlsx; do $SOFF \$f --outdir /work/out /work/figures.fods; done;
for f in odp ppt pptx; do $SOFF \$f --outdir /work/out /work/pitch.fodp; done;
$SOFF odt --outdir /work/out /work/form.fodt; $SOFF pdf --outdir /work/out /work/form.fodt; $SOFF pdf --outdir /work/out /work/pitch.fodp; $SOFF pdf --outdir /work/out /work/figures.fods" 2>&1 | grep -v "^$" | tail -40
