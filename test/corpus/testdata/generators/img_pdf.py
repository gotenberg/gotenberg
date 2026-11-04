import os, math, random
from PIL import Image, ImageDraw, ImageFont
C=os.environ["C"]; I=C+"/office/images"; P=C+"/pdf"
def font(sz):
    return ImageFont.truetype("/System/Library/Fonts/Supplemental/Arial Bold.ttf",sz)
random.seed(1)
W,H=1600,1200
im=Image.new("RGB",(W,H)); px=im.load()
for y in range(H):
    for x in range(W):
        px[x,y]=(int(255*x/W),int(255*y/H),int(255*(1-x/W)*0.7+60))
d=ImageDraw.Draw(im)
for _ in range(400):
    x,y=random.randrange(W),random.randrange(H); r=random.randrange(4,27)
    d.ellipse((x-r,y-r,x+r,y+r),outline=(random.randrange(256),random.randrange(256),random.randrange(256)),width=3)
d.text((W//2,H//2),"PHOTO",font=font(333),fill="white",anchor="mm")
im.save(I+"/photo.jpg",quality=85)
dg=Image.new("RGBA",(1200,900),(0,0,0,0)); d=ImageDraw.Draw(dg)
d.rectangle((100,100,700,500),fill=(255,0,0,128)); d.ellipse((400,300,1000,800),fill=(0,0,255,128)); d.polygon([(900,50),(1150,400),(650,400)],fill=(0,200,0,200)); d.text((600,850),"DIAGRAM",font=font(60),fill=(0,0,0,255),anchor="mm")
dg.save(I+"/diagram.png")
g=Image.new("P",(256,256)); g.putpalette([c for i in range(256) for c in (i,255-i,128)])
gd=ImageDraw.Draw(g)
for i in range(0,256,16): gd.rectangle((i,0,i+15,255),fill=i)
gd.ellipse((64,64,192,192),fill=255); g.save(I+"/icon.gif")
t=Image.new("L",(2000,2800),255); td=ImageDraw.Draw(t)
for y in range(0,2800,40): td.line((0,y,2000,y),fill=200)
td.text((1000,1400),"SCAN",font=font(400),fill=0,anchor="mm"); t.save(I+"/scan.tif",compression="tiff_lzw")
b=Image.new("RGB",(320,240),(240,240,100)); ImageDraw.Draw(b).text((160,120),"BMP",font=font(75),fill=(0,0,0),anchor="mm"); b.save(I+"/pic.bmp")
wp=Image.new("RGB",(800,600),(100,200,240)); ImageDraw.Draw(wp).text((400,300),"WEBP",font=font(150),fill=(0,0,0),anchor="mm"); wp.save(I+"/pic.webp",quality=90)
open(I+"/logo.svg","w").write('<svg xmlns="http://www.w3.org/2000/svg" width="300" height="200" viewBox="0 0 300 200"><rect x="10" y="10" width="120" height="80" fill="#c33"/><circle cx="210" cy="100" r="60" fill="#39c"/><text x="150" y="190" font-size="24" text-anchor="middle">SVG-MARKER</text></svg>\n')
s=Image.new("RGB",(600,200),"white"); ImageDraw.Draw(s).text((300,100),"CONFIDENTIAL",font=font(80),fill=(220,0,0),anchor="mm"); s.save(P+"/stamp.jpg",quality=92)
Image.new("RGB",(1,1),(0,0,0)).save(P+"/black.png")
open(P+"/attachment.csv","w").write("id,name,amount\n1,alpha,10\n2,beta,20\n3,gamma,30\n4,delta,40\n")
# ---- PDFs
from reportlab.pdfgen import canvas
from reportlab.lib.pagesizes import letter
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont
pdfmetrics.registerFont(TTFont("Arial","/System/Library/Fonts/Supplemental/Arial.ttf"))
pw,ph=letter
c=canvas.Canvas(P+"/report.pdf",pagesize=letter,pageCompression=0,initialFontName="Arial"); c.setTitle("Quarterly Report"); c.setAuthor("Finance")
for i in (1,2,3):
    c.setFont("Arial",24); c.drawString(72,720,"Quarterly Report")
    c.setFont("Arial",12)
    for k in range(10): c.drawString(72,680-k*16,f"Body line {k+1} of page {i}: revenue grew steadily across all regions this quarter.")
    c.setFont("Arial",36); c.drawString(72,400,f"REPORT-PAGE-{i}"); c.showPage()
c.save()
def simple(name,labels,fn=None):
    c=canvas.Canvas(P+"/"+name,pagesize=letter)
    for l in labels:
        c.setFont("Helvetica-Bold",36); c.drawString(72,400,l)
        if fn: fn(c)
        c.showPage()
    c.save()
simple("annex.pdf",["ANNEX-PAGE-1","ANNEX-PAGE-2"])
def lh(c): pass
c=canvas.Canvas(P+"/letterhead.pdf",pagesize=letter); c.setFillColorRGB(0.1,0.3,0.7); c.rect(0,ph-90,pw,90,fill=1,stroke=0); c.setFillColorRGB(1,1,1); c.setFont("Helvetica-Bold",30); c.drawString(72,ph-58,"NORTHWIND LETTERHEAD"); c.showPage(); c.save()
c=canvas.Canvas(P+"/transparent.pdf",pagesize=letter)
for i,(r,g,b) in enumerate([(1,0,0),(0,1,0),(0,0,1)]):
    c.setFillColorRGB(r,g,b,alpha=0.5); c.circle(250+i*60,500,100,fill=1,stroke=0)
c.setFillColorRGB(0,0,0,alpha=1); c.setFont("Helvetica-Bold",30); c.drawString(72,300,"TRANSPARENT-MARKER"); c.showPage(); c.save()
c=canvas.Canvas(P+"/colors.pdf",pagesize=letter)
c.setFillColorRGB(1,0,0); c.rect(72,600,100,60,fill=1); c.setFillColorRGB(0,1,0); c.rect(190,600,100,60,fill=1); c.setFillColorRGB(0,0,1); c.rect(308,600,100,60,fill=1)
c.setFillColorCMYK(1,0,0,0); c.rect(72,500,100,60,fill=1); c.setFillGray(0.5); c.rect(190,500,100,60,fill=1)
c.setFillColorRGB(0,0,0); c.setFont("Helvetica-Bold",30); c.drawString(72,300,"COLORS-MARKER"); c.showPage(); c.save()
c=canvas.Canvas(P+"/image-heavy.pdf",pagesize=letter)
for i in range(3):
    c.drawImage(I+"/photo.jpg",0,ph-(pw*0.75)-20,width=pw,height=pw*0.75); c.setFont("Helvetica",20); c.drawString(72,100,f"IMAGE-HEAVY-PAGE-{i+1}"); c.showPage()
c.save()
