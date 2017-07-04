module Main exposing (..)

import AnimationFrame
import Array exposing (Array, empty, toList)
import Keyboard exposing (..)
import Html exposing (..)
import Html.Attributes exposing (href)
import WebSocket
import Json.Encode exposing (Value, encode)
import Json.Decode exposing (decodeString, Decoder)
import Json.Decode.Pipeline exposing (decode, required)
import Svg exposing (..)
import Svg.Attributes exposing (..)
import Time exposing (Time)


main : Program Flags Model Msg
main =
    Html.programWithFlags
        { init = init
        , view = view
        , update = update
        , subscriptions = subscriptions
        }



-- MODEL


type alias S2CMessage =
    { players : Array Player
    , yourId : Int
    }


type alias Player =
    { x : Float
    , y : Float
    , a : Float
    , vx : Float
    , vy : Float
    , va : Float
    , r : Float
    , id : Int
    }


type alias C2SMessage =
    { l : Bool
    , r : Bool
    , u : Bool
    , d : Bool
    }


type alias Model =
    { flags : Flags
    , inputUser : String
    , inputBody : String
    , state : S2CMessage
    , stateFrom : Maybe Time
    , now : Maybe Time
    , keyState : C2SMessage
    }


type alias Flags =
    { endpoint : String }


init : Flags -> ( Model, Cmd Msg )
init flags =
    ( Model flags "" "" (S2CMessage empty -1) Nothing Nothing (C2SMessage False False False False), Cmd.none )


playerDecoder : Decoder Player
playerDecoder =
    decode Player
        |> required "x" Json.Decode.float
        |> required "y" Json.Decode.float
        |> required "a" Json.Decode.float
        |> required "vx" Json.Decode.float
        |> required "vy" Json.Decode.float
        |> required "va" Json.Decode.float
        |> required "r" Json.Decode.float
        |> required "id" Json.Decode.int


messageDecoder : Decoder S2CMessage
messageDecoder =
    decode S2CMessage
        |> required "players" (Json.Decode.array playerDecoder)
        |> required "yourid" Json.Decode.int


messageEncoder : C2SMessage -> Value
messageEncoder msg =
    Json.Encode.object
        [ ( "l", Json.Encode.bool msg.l )
        , ( "r", Json.Encode.bool msg.r )
        , ( "u", Json.Encode.bool msg.u )
        , ( "d", Json.Encode.bool msg.d )
        ]



-- UPDATE


type Msg
    = OnFrame Time
    | NewMessage String
    | KeyDown KeyCode
    | KeyUp KeyCode


leftKeyCode : Int
leftKeyCode =
    37


rightKeyCode : Int
rightKeyCode =
    39


upKeyCode : Int
upKeyCode =
    38


downKeyCode : Int
downKeyCode =
    40


changeKey : KeyCode -> Bool -> C2SMessage -> C2SMessage
changeKey key isDown state =
    if key == leftKeyCode then
        { state | l = isDown }
    else if key == rightKeyCode then
        { state | r = isDown }
    else if key == upKeyCode then
        { state | u = isDown }
    else if key == downKeyCode then
        { state | d = isDown }
    else
        state


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        OnFrame justNow ->
            case model.stateFrom of
                Just _ ->
                    ( { model | now = Just justNow }, Cmd.none )

                Nothing ->
                    ( { model | now = Just justNow, stateFrom = Just justNow }, Cmd.none )

        NewMessage str ->
            case decodeString messageDecoder str of
                Ok newState ->
                    ( { model | state = newState, now = Nothing, stateFrom = Nothing }, Cmd.none )

                Err err ->
                    Debug.crash err

        KeyDown code ->
            let
                newKeyState =
                    (changeKey code True model.keyState)
            in
                ( { model | keyState = newKeyState }
                , WebSocket.send model.flags.endpoint (encode 0 (messageEncoder newKeyState))
                )

        KeyUp code ->
            let
                newKeyState =
                    (changeKey code False model.keyState)
            in
                ( { model | keyState = newKeyState }
                , WebSocket.send model.flags.endpoint (encode 0 (messageEncoder newKeyState))
                )



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ AnimationFrame.times OnFrame
        , WebSocket.listen model.flags.endpoint NewMessage
        , downs KeyDown
        , ups KeyUp
        ]



-- VIEW


view : Model -> Html Msg
view model =
    div []
        [ svg [ width "320", height "320", viewBox "0 0 320 320" ]
            ((rect [ width "320", height "320", fill "none", stroke "#000" ] [])
                :: (List.concatMap
                        (viewPlayer model.state.yourId (diffTime model.stateFrom model.now))
                        (toList model.state.players)
                   )
            )
        , div []
            [ Html.text "Usage:"
            , br [] []
            , Html.text "Left and Right Key : rotation"
            , br [] []
            , Html.text "Up Key : accel"
            , br [] []
            , Html.text "Down Key : shoot bullets"
            , br [] []
            , br [] []
            , Html.a [ href "https://github.com/neguse/son" ] [ Html.text "Source code is available here." ]
            ]
        ]


diffTime : Maybe Time -> Maybe Time -> Time
diffTime begin now =
    case ( begin, now ) of
        ( Just jBegin, Just jNow ) ->
            jNow - jBegin

        _ ->
            0


reckoningPlayer : Time -> Player -> Player
reckoningPlayer t p =
    let
        tsec =
            Time.inSeconds t
    in
        { p | x = p.x + tsec * p.vx, y = p.y + tsec * p.vy, a = p.a + tsec * p.va }


viewPlayer : Int -> Time -> Player -> List (Svg msg)
viewPlayer yourId t p =
    let
        p_ =
            (reckoningPlayer t p)

        me =
            (p.id == yourId)

        myBullet =
            (p.id == -yourId)

        otherBullet =
            (p.id < 0)

        fillColor =
            if me then
                "#fff"
            else if myBullet then
                "#aaa"
            else if otherBullet then
                "#f22"
            else
                "#000"

        strokeColor =
            "#000"
    in
        [ circle
            [ cx (toString p_.x)
            , cy (toString p_.y)
            , r (toString p_.r)
            , stroke strokeColor
            , fill fillColor
            ]
            []
        , line
            [ x1 (toString p_.x)
            , y1 (toString p_.y)
            , x2 (toString (p_.x + p_.r * 2 * (cos p_.a)))
            , y2 (toString (p_.y + p_.r * 2 * (sin p_.a)))
            , stroke strokeColor
            ]
            []
        ]
