package app

// mapperAppsScript is a bound Google Apps Script receiver for the master radio
// dataset's exact 15-column Additions Queue schema. Users deploy it once as a
// web app, then paste its /exec URL and matching secret into GP-SDR.
const mapperAppsScript = `// GP-SDR -> Local Radio Channel Data / Additions Queue
// 1. Open the target Google Sheet, then Extensions > Apps Script.
// 2. Replace the editor contents with this file.
// 3. Change CHANGE_ME below to a private random value and use the same value in GP-SDR.
// 4. Deploy > New deployment > Web app; execute as yourself; allow anyone with the URL.
// 5. Paste the deployment /exec URL into GP-SDR Mapper > Google Sheet.

const GP_SDR_SECRET = 'CHANGE_ME';
const TARGET_SHEET = 'Additions Queue';
const COLUMNS = ['dateAdded','contributor','type','nameLabel','rxMHz','txMHz','mode','toneCode','locationSystem','whatWasHeard','dateTimeHeard','sourceURLFile','confidence','reviewStatus','reviewerNotes'];

function doPost(event) {
  const lock = LockService.getDocumentLock();
  try {
    const body = JSON.parse((event && event.postData && event.postData.contents) || '{}');
    if (!GP_SDR_SECRET || GP_SDR_SECRET === 'CHANGE_ME' || body.secret !== GP_SDR_SECRET) return reply({ok:false,error:'Unauthorized'});
    if (body.sheetName && body.sheetName !== TARGET_SHEET) return reply({ok:false,error:'Wrong target sheet'});
    const rows = Array.isArray(body.rows) ? body.rows : [];
    if (!rows.length) return reply({ok:true,added:0});
    lock.waitLock(10000);
    const sheet = SpreadsheetApp.getActiveSpreadsheet().getSheetByName(TARGET_SHEET);
    if (!sheet) throw new Error('Missing Additions Queue tab');
    const headers = sheet.getRange(4, 1, 1, 15).getDisplayValues()[0];
    const expected = ['Date Added','Contributor','Type','Name/Label','RX MHz','TX MHz','Mode','Tone/Code','Location/System','What Was Heard','Date/Time Heard','Source URL/File','Confidence','Review Status','Reviewer Notes'];
    if (headers.join('|') !== expected.join('|')) throw new Error('Additions Queue columns do not match the GP-SDR master schema');
    const values = rows.map(row => COLUMNS.map(key => safeCell(row[key])));
    const startRow = Math.max(5, sheet.getLastRow() + 1);
    sheet.getRange(startRow, 1, values.length, 15).setValues(values);
    return reply({ok:true,added:values.length,startRow:startRow});
  } catch (error) {
    return reply({ok:false,error:String(error && error.message || error)});
  } finally {
    try { lock.releaseLock(); } catch (_) {}
  }
}

function safeCell(value) {
  if (value === null || value === undefined) return '';
  if (typeof value === 'number' || typeof value === 'boolean') return value;
  const text = String(value);
  return /^[=+\-@]/.test(text) ? "'" + text : text;
}

function reply(value) {
  return ContentService.createTextOutput(JSON.stringify(value)).setMimeType(ContentService.MimeType.JSON);
}
`
