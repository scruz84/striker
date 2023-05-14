import SoccerLineUp from 'react-soccer-lineup'
import Box from "@mui/material/Box";
import "./teamview.css";

export function MatchPlayersDetails() {
    let homeTeam = {
        color: "red",
        squad: {
            cam: [
                { name: "Hicham" },
                { player: { number: 8 } },
                { player: { number: 6 } },
                { player: { number: 10 } }
            ],
            df: [{ number: 2 }, { number: 4 }, { number: 5 }, { number: 3 }],
            fw: [{ number: 9 }, { number: 11 }],
            gk: { number: 1 }
        }
    };

    let awayTeam = {
        color: "red",
        squad: {
            gk: { number: 1 , name: "Luis Abel Baraldes" },
            df: [
                { number: 2, name: "Luis Abel Baraldes" },
                { number: 4, name: "Luis Abel Baraldes"  },
                { number: 5 , name: "Luis Abel Baraldes" },
                { number: 3 , name: "Luis Abel Baraldes" }
            ],
            cam: [{ number: 7 , name: "Luis Abel Baraldes" }, { number: 8 , name: "Luis Abel Baraldes" }, { number: 6 , name: "Luis Abel Baraldes" }],
            fw: [{ number: 9 , name: "Luis Abel Baraldes" }, { number: 11 , name: "Luis Abel Baraldes" }, { number: 10, name: "Luis Abel Baraldes"  }]
        }
    };

    return (
        <Box>
            <SoccerLineUp
                size={ "responsive" }
                color={ "#40823d" }
                pattern={ "lines" }
                awayTeam={awayTeam}
                homeTeam={homeTeam}
            />
        </Box>
    )
}
