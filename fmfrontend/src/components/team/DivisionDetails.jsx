import {useParams} from "react-router-dom";
import {useDispatch, useSelector} from "react-redux";
import {useEffect} from "react";
import {GetDivisionDetails} from "../user/_divisionFunctions";
import {Paper, Table, TableBody, TableCell, TableContainer, TableHead, TableRow} from "@mui/material";
import Box from "@mui/material/Box";

export function DivisionDetails() {
    const routeParams = useParams();

    const dispatch = useDispatch()
    const divReducer = useSelector((state) => state.userReducer);
    const divisionDetails = divReducer.divisionDetails;

    useEffect(() => {
        // Anything in here is fired on component mount.
        if (!divisionDetails) {
            GetDivisionDetails(dispatch, routeParams.id)
        }
        return () => {
            // Anything in here is fired on component unmount.
        }
    }, [])

    if (divReducer.divisionDetails) {
        const classificationTable = GetClassificationTable(divisionDetails)
        return (
            <div>
                <h3>Jornada {divisionDetails.classification.positions[0].matches}</h3>
                <Box sx={{ width: 800}}>
                    {classificationTable}
                </Box>
            </div>
        );
    } else {
        return ( <div>loading...</div> );
    }
}

function GetClassificationTable(divisionDetails) {

    return (
        <TableContainer component={Paper}>
            <Table sx={{ minWidth: 250 }} size="small" aria-label="Classification">
                <TableHead>
                    <TableRow>
                        <TableCell>#</TableCell>
                        <TableCell align="left">Equipo</TableCell>
                        <TableCell align="left">Puntos</TableCell>
                        <TableCell align="left">Avg</TableCell>
                        <TableCell align="left">G</TableCell>
                        <TableCell align="left">E</TableCell>
                        <TableCell align="left">P</TableCell>
                        <TableCell align="left">+</TableCell>
                        <TableCell align="left">-</TableCell>
                    </TableRow>
                </TableHead>
                <TableBody>
                    {divisionDetails.classification.positions.map((row) => (
                        <TableRow
                            key={row.position}
                            sx={{ '&:last-child td, &:last-child th': { border: 0 } }}>
                            <TableCell component="th" scope="row">{row.position}</TableCell>
                            <TableCell align="left">{row.teamName}</TableCell>
                            <TableCell align="center">{row.points}</TableCell>
                            <TableCell align="left">{row.goalsScored-row.goalsReceived}</TableCell>
                            <TableCell align="left">{row.wins}</TableCell>
                            <TableCell align="left">{row.draws}</TableCell>
                            <TableCell align="left">{row.losses}</TableCell>
                            <TableCell align="left">{row.goalsScored}</TableCell>
                            <TableCell align="left">{row.goalsReceived}</TableCell>
                        </TableRow>
                    ))}
                </TableBody>
            </Table>
        </TableContainer>
    );
}