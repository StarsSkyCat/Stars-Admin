package table

const decimalHelper = `
var decimalPattern=regexp.MustCompile("^-?[0-9]+(?:[.][0-9]+)?$")
func validDecimal(value string,precision,scale int)bool{if len(value)>4096||!decimalPattern.MatchString(value){return false};value=strings.TrimPrefix(value,"-");whole,fraction,_:=strings.Cut(value,".");if scale>=0&&len(fraction)>scale{return false};if precision>0{whole=strings.TrimLeft(whole,"0");if scale>=0&&len(whole)>precision-scale{return false};if len(whole)+len(fraction)>precision{return false}};return true}
`
const filterHelper = `
func parseFilter(text,kind string,maxLength,precision,scale int)(any,error){switch kind{case "id":return httpx.ParseID(text);case "int2","int4","int8":bits:=64;switch kind{case "int2":bits=16;case "int4":bits=32};value,err:=strconv.ParseInt(text,10,bits);if err!=nil{return nil,response.ErrInvalidParam};return value,nil;case "bool":if text!="true"&&text!="false"{return nil,response.ErrInvalidParam};value,_:=strconv.ParseBool(text);return value,nil;case "timestamp":value,err:=time.Parse(time.RFC3339,text);if err!=nil{return nil,response.ErrInvalidParam};return value.UTC(),nil;case "numeric":if !validDecimal(text,precision,scale){return nil,response.ErrInvalidParam};return text,nil;default:if utf8.RuneCountInString(text)>maxLength{return nil,response.ErrInvalidParam};return text,nil}}
`
