import os, json, csv, datetime, copy
from docx import Document
from docx.shared import Inches, Pt
from docx.oxml import parse_xml
from docx.oxml.ns import nsdecls, qn
C=os.environ["C"]; B=os.environ["B"]; O=C+"/office"
# ---- contract
d=Document(); cp=d.core_properties; cp.title="Master Services Agreement"; cp.author="Legal Department"; cp.subject="Services"; cp.keywords="contract, msa"
sec=d.sections[0]
sec.header.paragraphs[0].text="Master Services Agreement"
fp=sec.footer.paragraphs[0]; fp.add_run("Page ")
r=fp.add_run(); r._r.append(parse_xml(f'<w:fldSimple {nsdecls("w")} w:instr=" PAGE "><w:r><w:t>1</w:t></w:r></w:fldSimple>'))
d.add_paragraph("Master Services Agreement",style="Title")
d.add_paragraph("between Northwind Traders (the Provider) and Contoso Ltd (the Client).")
d.add_picture(C+"/web/invoice/logo.png",width=Inches(2))
names=["Definitions","Scope of Services","Fees and Payment","Term and Termination","Confidentiality","Liability","Intellectual Property","Governing Law"]
prose="The parties agree that this clause shall be interpreted in good faith and in accordance with the commercial purpose of the Agreement. Each party shall perform its obligations with reasonable skill and care, and shall notify the other promptly of any matter likely to affect performance. "
for i,n in enumerate(names,1):
    d.add_heading(f"{i}. {n}",level=1)
    for j in (1,2):
        d.add_heading(f"{i}.{j} {n} - sub-clause {j}",level=2)
        for _ in range(3): d.add_paragraph(prose*2)
    if i==1:
        t=d.add_table(rows=4,cols=3); t.style="Table Grid"
        for c,h in enumerate(["Term","Meaning","Section"]): t.cell(0,c).text=h
        for r_,(a,b) in enumerate([("Services","The services described in Schedule A"),("Fees","Amounts payable under clause 3"),("Confidential Information","Non-public information")],1):
            t.cell(r_,0).text=a; t.cell(r_,1).text=b; t.cell(r_,2).text=str(r_)
    if i==2:
        for b in ["Consulting services","Hosting services","Support services"]: d.add_paragraph(b,style="List Bullet")
d.add_page_break()
d.add_heading("Schedule A",level=1)
d.add_paragraph("Schedule A lists the services and the applicable service levels.")
d.save(O+"/contract.docx")
# ---- i18n
L=json.load(open(B+"/i18n.json"))
d=Document(); d.add_heading("Multilingual document",1)
for m,t,dr in L:
    p=d.add_paragraph(f"{m}: {t}")
    if dr=="rtl": p._p.get_or_add_pPr().append(parse_xml(f'<w:bidi {nsdecls("w")}/>'))
d.save(O+"/i18n.docx")
# ---- tracked
d=Document(); p=d.add_paragraph("Start of sentence. ")
W=nsdecls("w")
p._p.append(parse_xml(f'<w:ins {W} w:id="1" w:author="Jane Doe" w:date="2026-01-15T10:00:00Z"><w:r><w:t>INSERTED-TEXT</w:t></w:r></w:ins>'))
p._p.append(parse_xml(f'<w:del {W} w:id="2" w:author="John Smith" w:date="2026-01-16T11:00:00Z"><w:r><w:delText>DELETED-TEXT</w:delText></w:r></w:del>'))
p.add_run(" End of sentence.")
d.save(O+"/tracked.docx")
# ---- xlsx
from openpyxl import Workbook
from openpyxl.styles import Font, PatternFill
wb=Workbook(); s=wb.active; s.title="Summary"
s.merge_cells("A1:D1"); s["A1"]="FY2026 Budget"; s["A1"].font=Font(bold=True,size=16)
for c,h in enumerate(["Category","H1","H2","Total"],1): s.cell(3,c,h)
for r,(n,a,b) in enumerate([("Cloud hosting",120000,130000),("Salaries",800000,820000),("Marketing",90000,100000)],4):
    s.cell(r,1,n); s.cell(r,2,a); s.cell(r,3,b); s.cell(r,4,f"=SUM(B{r}:C{r})")
s["A7"]="Total"; 
for c in "BCD": s[f"{c}7"]=f"=SUM({c}4:{c}6)"
m=wb.create_sheet("Monthly")
mon=["Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec"]
hdr=["Category"]+mon+["Total"]
for c,h in enumerate(hdr,1):
    x=m.cell(1,c,h); x.font=Font(bold=True); x.fill=PatternFill("solid",fgColor="DDDDDD")
cats=["Cloud hosting","Salaries","Marketing","Travel","Software licenses","Office rent","Training","Legal","Insurance","Utilities"]
for r in range(2,62):
    cat=cats[(r-2)%10] if r>11 else cats[r-2]
    m.cell(r,1,cat if r<=11 else f"{cat} {r-1}")
    for c in range(2,14):
        m.cell(r,c,round(1000+r*17.5+c*3.25,2)).number_format="#,##0.00"
    m.cell(r,14,f"=SUM(B{r}:M{r})").number_format="#,##0.00"
m.freeze_panes="B2"; m.page_setup.orientation="landscape"; m.print_title_rows="1:1"
n=wb.create_sheet("Notes")
for i,t in enumerate(["Budget notes","Assumptions: 3% inflation","Prepared by Finance"],1): n.cell(i,1,t)
wb.save(O+"/budget.xlsx")
# ---- csv
with open(O+"/sales.csv","w",encoding="utf-8",newline="") as f:
    w=csv.writer(f); w.writerow(["id","date","customer","product","qty","unit_price","total","comment"])
    cust=["Café Müller","Straße GmbH","Élodie & Fils","Northwind Traders","Contoso Ltd"]
    for i in range(1,26):
        q=i%7+1; u=round(9.99+i*1.37,2)
        w.writerow([i,(datetime.date(2026,1,1)+datetime.timedelta(days=i*9)).isoformat(),cust[i%5],f"Widget {i}",q,u,round(q*u,2),f"Note, with comma #{i}" if i%3==0 else "ok"])
# ---- pptx
from pptx import Presentation
from pptx.util import Inches as I
from pptx.chart.data import CategoryChartData
from pptx.enum.chart import XL_CHART_TYPE
P=Presentation(); P.slide_width=I(13.333); P.slide_height=I(7.5)
def ts(title,body=None,notes=None):
    s=P.slides.add_slide(P.slide_layouts[1 if body else 5]); s.shapes.title.text=title
    if body: s.placeholders[1].text="\n".join(body)
    if notes: s.notes_slide.notes_text_frame.text=notes
    return s
s=P.slides.add_slide(P.slide_layouts[0]); s.shapes.title.text="Q4 Product Review"; s.placeholders[1].text="Product team"
ts("Agenda",["Results","Roadmap","Risks"],"NOTES-MARKER-2")
s=ts("Two columns"); 
for x,t in [(0.5,"Left column text"),(6.9,"Right column text")]:
    tb=s.shapes.add_textbox(I(x),I(1.8),I(6),I(3)); tb.text_frame.text=t
s=ts("Table"); tt=s.shapes.add_table(3,3,I(1),I(2),I(10),I(2)).table
for r in range(3):
    for c in range(3): tt.cell(r,c).text=f"R{r}C{c}"
s=ts("Bar chart"); cd=CategoryChartData(); cd.categories=["Q1","Q2","Q3","Q4"]; cd.add_series("Revenue",(10,14,12,18))
s.shapes.add_chart(XL_CHART_TYPE.COLUMN_CLUSTERED,I(1.5),I(1.8),I(10),I(5),cd)
s=ts("Image"); s.shapes.add_picture(C+"/web/invoice/logo.png",I(4),I(2.5),I(5))
s=ts("Hidden slide",["HIDDEN-SLIDE-MARKER"]); s._element.set("show","0")
ts("Thank you",None,"NOTES-MARKER-8")
P.save(O+"/deck.pptx")
# ---- txt, xml, htm
lines=[f"Line {i:02d}: Café résumé naïve déjà vu — "+"The quick brown fox jumps over the lazy dog. "*4 for i in range(1,61)]
open(O+"/readme.txt","w",encoding="utf-8").write("\n".join(lines)+"\n")
open(O+"/data.xml","w",encoding="utf-8").write('<?xml version="1.0" encoding="UTF-8"?>\n<catalog><item id="1"><name>XML-MARKER</name></item><item id="2"><name>Second</name></item></catalog>\n')
open(O+"/page.htm","w",encoding="utf-8").write('<!doctype html><html><head><meta charset="utf-8"><title>Page</title></head><body><h1>HTM-MARKER</h1><h2>Section</h2><table border="1"><tr><th>A</th><th>B</th></tr><tr><td>1</td><td>2</td></tr></table></body></html>\n')
# ---- rtf
open(O+"/letter.rtf","w").write(r"""{\rtf1\ansi\deff0{\fonttbl{\f0 Arial;}}
\pard\b\fs32 Northwind Traders\b0\fs22\par
1 Harbour Road, Seattle\par
\par
November 3, 2026\par
\par
Dear Contoso team,\par
This letter is \b bold\b0  and \i italic\i0  in places.\par
\pard{\pntext\f0\'B7\tab}{\*\pn\pnlvlblt\pnf1\pnindent360{\pntxtb\'B7}}\fi-360\li720 First item\par
{\pntext\f0\'B7\tab}Second item\par
\pard\par
Sincerely,\par
Jane Doe\par
}
""")
